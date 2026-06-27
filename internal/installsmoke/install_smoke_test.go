package installsmoke

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cloud-byte-consulting/cachy/internal/release"
)

func TestArchiveInstallSmokeDoctorAndProxyHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("install smoke test builds and starts a real Cachy binary")
	}

	root := t.TempDir()
	binaryPath := filepath.Join(root, installedBinaryName())
	build := exec.Command("go", "build", "-trimpath", "-o", binaryPath, "./cmd/cachy")
	build.Dir = repoRoot(t)
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}

	artifact, err := release.PackageArchive(release.ArchiveOptions{
		BinaryPath: binaryPath,
		OutputDir:  filepath.Join(root, "dist"),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	})
	if err != nil {
		t.Fatalf("PackageArchive() error = %v", err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() bin error = %v", err)
	}
	extracted := filepath.Join(binDir, installedBinaryName())
	if err := extractArchive(artifact.ArchivePath, extracted); err != nil {
		t.Fatalf("extractArchive() error = %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cachy := lookupCachyOnPath(t)
	targetURL, closeTarget := startTargetServer(t)
	defer closeTarget()

	doctor := exec.Command(cachy, "doctor", "--target", targetURL, "--listen", "127.0.0.1:0")
	doctorOutput, err := doctor.CombinedOutput()
	if err != nil {
		t.Fatalf("cachy doctor failed: %v\n%s", err, doctorOutput)
	}
	if !strings.Contains(string(doctorOutput), "OK target-url") || !strings.Contains(string(doctorOutput), "OK listen-port") {
		t.Fatalf("doctor output missing expected checks:\n%s", doctorOutput)
	}

	listen := "127.0.0.1:" + freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	proxy := exec.CommandContext(ctx, cachy, "proxy", "--listen", listen, "--target", targetURL)
	if err := proxy.Start(); err != nil {
		t.Fatalf("cachy proxy start failed: %v", err)
	}
	defer func() {
		cancel()
		_ = proxy.Wait()
	}()

	assertHealthz(t, "http://"+listen+"/healthz")
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func installedBinaryName() string {
	if runtime.GOOS == "windows" {
		return "cachy.exe"
	}
	return "cachy"
}

func extractArchive(archivePath, outputPath string) error {
	switch {
	case strings.HasSuffix(archivePath, ".zip"):
		return extractZip(archivePath, outputPath)
	case strings.HasSuffix(archivePath, ".tar.gz"):
		return extractTarGz(archivePath, outputPath)
	default:
		return fmt.Errorf("unsupported archive %s", archivePath)
	}
}

func extractZip(archivePath, outputPath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) != 1 {
		return fmt.Errorf("zip entries = %d, want 1", len(reader.File))
	}
	entry, err := reader.File[0].Open()
	if err != nil {
		return err
	}
	defer entry.Close()
	return writeExtracted(outputPath, entry, 0o755)
}

func extractTarGz(archivePath, outputPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	header, err := tr.Next()
	if err != nil {
		return err
	}
	return writeExtracted(outputPath, tr, os.FileMode(header.Mode))
}

func writeExtracted(outputPath string, input io.Reader, mode os.FileMode) error {
	output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, input)
	return err
}

func lookupCachyOnPath(t *testing.T) string {
	t.Helper()

	cachy, err := exec.LookPath("cachy")
	if err != nil {
		t.Fatalf("cachy was not resolved through PATH: %v", err)
	}
	return cachy
}

func startTargetServer(t *testing.T) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() target error = %v", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("target ok"))
	})}
	go func() { _ = server.Serve(listener) }()
	return "http://" + listener.Addr().String(), func() {
		_ = server.Close()
	}
}

func freePort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() free port error = %v", err)
	}
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort() error = %v", err)
	}
	return port
}

func assertHealthz(t *testing.T, url string) {
	t.Helper()

	client := http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(10 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr == nil && resp.StatusCode == http.StatusOK && strings.TrimSpace(string(body)) == "ok" {
				return
			}
			lastErr = fmt.Errorf("status=%d body=%q readErr=%v", resp.StatusCode, body, readErr)
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("health check %s did not become ready: %v", url, lastErr)
}
