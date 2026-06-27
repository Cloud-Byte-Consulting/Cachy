package releaseworkflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseWorkflowBuildsSixRequiredCLITargets(t *testing.T) {
	t.Parallel()

	workflow, err := os.ReadFile(filepath.Join("..", "..", ".gitea", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("release workflow is required: %v", err)
	}
	text := string(workflow)

	requiredSnippets := []string{
		"workflow_dispatch:",
		"tags:",
		"'v*'",
		"CGO_ENABLED: \"0\"",
		"go build -trimpath",
		"./cmd/cachy",
		"go run ./cmd/cachy-release package",
		"go run ./cmd/cachy-release sbom",
		"go run ./cmd/cachy-release notes",
		".sha256",
		".sbom.json",
		"RELEASE_NOTES.md",
		"actions/upload-artifact",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("release workflow missing %q", snippet)
		}
	}

	targets := []struct {
		goos   string
		goarch string
	}{
		{goos: "windows", goarch: "amd64"},
		{goos: "windows", goarch: "arm64"},
		{goos: "darwin", goarch: "amd64"},
		{goos: "darwin", goarch: "arm64"},
		{goos: "linux", goarch: "amd64"},
		{goos: "linux", goarch: "arm64"},
	}
	for _, target := range targets {
		t.Run(target.goos+"_"+target.goarch, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(text, "goos: "+target.goos) {
				t.Fatalf("release workflow missing GOOS %q", target.goos)
			}
			if !strings.Contains(text, "goarch: "+target.goarch) {
				t.Fatalf("release workflow missing GOARCH %q", target.goarch)
			}
			artifact := "cachy_" + target.goos + "_" + target.goarch
			if !strings.Contains(text, artifact) {
				t.Fatalf("release workflow missing artifact %q", artifact)
			}
		})
	}
}
