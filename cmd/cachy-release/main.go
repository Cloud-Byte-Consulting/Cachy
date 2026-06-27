package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cloud-byte-consulting/cachy/internal/release"
)

func main() {
	if len(os.Args) < 2 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: cachy-release <package|sbom|notes> ...")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "package":
		runPackage(os.Args[2:])
	case "sbom":
		runSBOM(os.Args[2:])
	case "notes":
		runNotes(os.Args[2:])
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}

func runPackage(args []string) {
	flags := flag.NewFlagSet("cachy-release package", flag.ExitOnError)
	input := flags.String("input", "", "built binary path")
	goos := flags.String("goos", "", "target GOOS")
	goarch := flags.String("goarch", "", "target GOARCH")
	output := flags.String("out", "dist", "output directory")
	if err := flags.Parse(args); err != nil {
		os.Exit(2)
	}

	artifact, err := release.PackageArchive(release.ArchiveOptions{
		BinaryPath: *input,
		OutputDir:  *output,
		GOOS:       *goos,
		GOARCH:     *goarch,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "package failed: %v\n", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "archive: %s\nchecksum: %s\n", artifact.ArchivePath, artifact.ChecksumPath)
}

func runSBOM(args []string) {
	flags := flag.NewFlagSet("cachy-release sbom", flag.ExitOnError)
	kind := flags.String("kind", "", "sbom kind")
	artifact := flags.String("artifact", "", "artifact path")
	output := flags.String("out", "", "output sbom path")
	version := flags.String("version", "", "release version")
	if err := flags.Parse(args); err != nil {
		os.Exit(2)
	}
	result, err := release.GenerateSBOM(release.SBOMOptions{
		Kind:       *kind,
		Artifact:   *artifact,
		OutputPath: *output,
		Version:    *version,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "sbom failed: %v\n", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "sbom: %s\n", result.OutputPath)
}

func runNotes(args []string) {
	flags := flag.NewFlagSet("cachy-release notes", flag.ExitOnError)
	version := flags.String("version", "", "release version")
	output := flags.String("out", "", "output notes path")
	if err := flags.Parse(args); err != nil {
		os.Exit(2)
	}
	result, err := release.GenerateReleaseNotes(release.ReleaseNotesOptions{
		Version: *version,
		Output:  *output,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "notes failed: %v\n", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "notes: %s\n", result.OutputPath)
}
