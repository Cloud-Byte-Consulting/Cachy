package releaseworkflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerfileDocumentsHeadlessRuntimeContract(t *testing.T) {
	t.Parallel()

	dockerfile, err := os.ReadFile(filepath.Join("..", "..", "Dockerfile"))
	if err != nil {
		t.Fatalf("Dockerfile is required: %v", err)
	}
	text := string(dockerfile)

	requiredSnippets := []string{
		"FROM --platform=$BUILDPLATFORM golang:1.24",
		"CGO_ENABLED=0",
		"GOOS=$TARGETOS",
		"GOARCH=$TARGETARCH",
		"go build -trimpath",
		"./cmd/cachy",
		"FROM alpine:",
		"adduser",
		"USER cachy",
		"EXPOSE 8787",
		"HEALTHCHECK",
		"http://127.0.0.1:8787/healthz",
		"ENTRYPOINT [\"/usr/local/bin/cachy\"]",
		"CMD [\"proxy\"]",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("Dockerfile missing %q", snippet)
		}
	}
}

func TestReleaseWorkflowBuildsMultiArchDockerArtifactWithoutPublishing(t *testing.T) {
	t.Parallel()

	workflow, err := os.ReadFile(filepath.Join("..", "..", ".gitea", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("release workflow is required: %v", err)
	}
	text := string(workflow)

	requiredSnippets := []string{
		"docker/setup-qemu-action",
		"docker/setup-buildx-action",
		"docker/build-push-action",
		"platforms: linux/amd64,linux/arm64",
		"push: false",
		"outputs: type=oci,dest=dist/cachy_linux_multiarch.oci.tar",
		"cachy_linux_multiarch.oci.tar",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("release workflow missing %q", snippet)
		}
	}
}
