package release

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageArchiveCreatesWindowsZipWithChecksum(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "cachy.exe")
	if err := os.WriteFile(input, []byte("windows binary"), 0o600); err != nil {
		t.Fatalf("WriteFile() input error = %v", err)
	}

	artifact, err := PackageArchive(ArchiveOptions{
		BinaryPath: input,
		OutputDir:  filepath.Join(root, "dist"),
		GOOS:       "windows",
		GOARCH:     "amd64",
	})
	if err != nil {
		t.Fatalf("PackageArchive() error = %v", err)
	}

	if filepath.Base(artifact.ArchivePath) != "cachy_windows_amd64.zip" {
		t.Fatalf("archive = %q, want cachy_windows_amd64.zip", filepath.Base(artifact.ArchivePath))
	}
	if filepath.Base(artifact.ChecksumPath) != "cachy_windows_amd64.zip.sha256" {
		t.Fatalf("checksum = %q, want cachy_windows_amd64.zip.sha256", filepath.Base(artifact.ChecksumPath))
	}
	assertChecksumFile(t, artifact.ArchivePath, artifact.ChecksumPath)
	assertZipContains(t, artifact.ArchivePath, "cachy.exe", "windows binary")
}

func TestPackageArchiveCreatesUnixTarGzWithExecutableModeAndChecksum(t *testing.T) {
	t.Parallel()

	for _, target := range []struct {
		goos   string
		goarch string
	}{
		{goos: "darwin", goarch: "arm64"},
		{goos: "linux", goarch: "amd64"},
	} {
		t.Run(target.goos+"_"+target.goarch, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			input := filepath.Join(root, "cachy")
			if err := os.WriteFile(input, []byte(target.goos+" binary"), 0o600); err != nil {
				t.Fatalf("WriteFile() input error = %v", err)
			}

			artifact, err := PackageArchive(ArchiveOptions{
				BinaryPath: input,
				OutputDir:  filepath.Join(root, "dist"),
				GOOS:       target.goos,
				GOARCH:     target.goarch,
			})
			if err != nil {
				t.Fatalf("PackageArchive() error = %v", err)
			}

			wantName := "cachy_" + target.goos + "_" + target.goarch + ".tar.gz"
			if filepath.Base(artifact.ArchivePath) != wantName {
				t.Fatalf("archive = %q, want %s", filepath.Base(artifact.ArchivePath), wantName)
			}
			assertChecksumFile(t, artifact.ArchivePath, artifact.ChecksumPath)
			assertTarGzContainsExecutable(t, artifact.ArchivePath, "cachy", target.goos+" binary")
		})
	}
}

func TestPackageArchiveRejectsUnsupportedTarget(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "cachy")
	if err := os.WriteFile(input, []byte("binary"), 0o600); err != nil {
		t.Fatalf("WriteFile() input error = %v", err)
	}

	_, err := PackageArchive(ArchiveOptions{
		BinaryPath: input,
		OutputDir:  filepath.Join(root, "dist"),
		GOOS:       "plan9",
		GOARCH:     "amd64",
	})
	if err == nil {
		t.Fatal("PackageArchive() succeeded, want unsupported target error")
	}
}

func assertChecksumFile(t *testing.T, archivePath, checksumPath string) {
	t.Helper()

	archive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile() archive error = %v", err)
	}
	sum := sha256.Sum256(archive)
	expected := hex.EncodeToString(sum[:]) + "  " + filepath.Base(archivePath)

	checksum, err := os.ReadFile(checksumPath)
	if err != nil {
		t.Fatalf("ReadFile() checksum error = %v", err)
	}
	if strings.TrimSpace(string(checksum)) != expected {
		t.Fatalf("checksum = %q, want %q", strings.TrimSpace(string(checksum)), expected)
	}
}

func assertZipContains(t *testing.T, archivePath, name, content string) {
	t.Helper()

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("OpenReader() zip error = %v", err)
	}
	defer reader.Close()

	if len(reader.File) != 1 {
		t.Fatalf("zip entries = %d, want 1", len(reader.File))
	}
	file := reader.File[0]
	if file.Name != name {
		t.Fatalf("zip entry = %q, want %q", file.Name, name)
	}
	rc, err := file.Open()
	if err != nil {
		t.Fatalf("Open() zip entry error = %v", err)
	}
	defer rc.Close()
	buf, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("Read() zip entry error = %v", err)
	}
	if string(buf) != content {
		t.Fatalf("zip content = %q, want %q", string(buf), content)
	}
}

func assertTarGzContainsExecutable(t *testing.T, archivePath, name, content string) {
	t.Helper()

	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Open() tar.gz error = %v", err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("NewReader() gzip error = %v", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	header, err := tr.Next()
	if err != nil {
		t.Fatalf("Next() tar entry error = %v", err)
	}
	if header.Name != name {
		t.Fatalf("tar entry = %q, want %q", header.Name, name)
	}
	if header.Mode != 0o755 {
		t.Fatalf("tar entry mode = %#o, want 0755", header.Mode)
	}
	buf, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("Read() tar entry error = %v", err)
	}
	if string(buf) != content {
		t.Fatalf("tar content = %q, want %q", string(buf), content)
	}
	if _, err := tr.Next(); err == nil {
		t.Fatal("tar has extra entries")
	}
}
