package release

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type ArchiveOptions struct {
	BinaryPath string
	OutputDir  string
	GOOS       string
	GOARCH     string
}

type ArchiveArtifact struct {
	ArchivePath  string
	ChecksumPath string
}

func PackageArchive(options ArchiveOptions) (ArchiveArtifact, error) {
	if options.BinaryPath == "" {
		return ArchiveArtifact{}, errors.New("binary path is required")
	}
	if options.OutputDir == "" {
		return ArchiveArtifact{}, errors.New("output dir is required")
	}

	format, err := archiveFormat(options.GOOS, options.GOARCH)
	if err != nil {
		return ArchiveArtifact{}, err
	}
	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return ArchiveArtifact{}, err
	}

	name := archiveName(options.GOOS, options.GOARCH, format)
	archivePath := filepath.Join(options.OutputDir, name)
	switch format {
	case "zip":
		err = writeZip(archivePath, options.BinaryPath, "cachy.exe")
	case "tar.gz":
		err = writeTarGz(archivePath, options.BinaryPath, "cachy")
	default:
		err = fmt.Errorf("unsupported archive format %q", format)
	}
	if err != nil {
		return ArchiveArtifact{}, err
	}

	checksumPath := archivePath + ".sha256"
	if err := writeChecksum(archivePath, checksumPath); err != nil {
		return ArchiveArtifact{}, err
	}
	return ArchiveArtifact{ArchivePath: archivePath, ChecksumPath: checksumPath}, nil
}

func archiveFormat(goos, goarch string) (string, error) {
	switch goos + "/" + goarch {
	case "windows/amd64", "windows/arm64":
		return "zip", nil
	case "darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64":
		return "tar.gz", nil
	default:
		return "", fmt.Errorf("unsupported release target %s/%s", goos, goarch)
	}
}

func archiveName(goos, goarch, format string) string {
	return fmt.Sprintf("cachy_%s_%s.%s", goos, goarch, format)
}

func writeZip(archivePath, binaryPath, entryName string) error {
	archive, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	zw := zip.NewWriter(archive)
	defer zw.Close()

	header := &zip.FileHeader{Name: entryName, Method: zip.Deflate}
	header.SetMode(0o755)
	entry, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	return copyFile(entry, binaryPath)
}

func writeTarGz(archivePath, binaryPath, entryName string) error {
	input, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}

	archive, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	gw := gzip.NewWriter(archive)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	header := &tar.Header{
		Name:    entryName,
		Mode:    0o755,
		Size:    info.Size(),
		ModTime: time.Unix(0, 0),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(tw, input)
	return err
}

func copyFile(dst io.Writer, path string) error {
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()
	_, err = io.Copy(dst, input)
	return err
}

func writeChecksum(archivePath, checksumPath string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, archive); err != nil {
		return err
	}
	line := fmt.Sprintf("%s  %s\n", hex.EncodeToString(hash.Sum(nil)), filepath.Base(archivePath))
	return os.WriteFile(checksumPath, []byte(line), 0o644)
}
