package release

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateSBOMWritesArtifactEvidence(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifact := filepath.Join(root, "cachy_linux_amd64.tar.gz")
	if err := os.WriteFile(artifact, []byte("archive bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile() artifact error = %v", err)
	}
	output := filepath.Join(root, "cachy_linux_amd64.tar.gz.sbom.json")

	result, err := GenerateSBOM(SBOMOptions{
		Kind:        "binary",
		Artifact:    artifact,
		OutputPath:  output,
		Version:     "v0.1.0",
		GeneratedAt: time.Unix(100, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("GenerateSBOM() error = %v", err)
	}
	if result.OutputPath != output {
		t.Fatalf("output = %q, want %q", result.OutputPath, output)
	}

	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile() sbom error = %v", err)
	}
	var document SBOMDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("Unmarshal() sbom error = %v", err)
	}
	if document.Name != "cachy" || document.Version != "v0.1.0" || document.Kind != "binary" {
		t.Fatalf("document identity = %#v", document)
	}
	if document.Artifact.Name != "cachy_linux_amd64.tar.gz" || document.Artifact.SizeBytes != int64(len("archive bytes")) {
		t.Fatalf("artifact = %#v, want archive metadata", document.Artifact)
	}
	if document.Artifact.SHA256 == "" {
		t.Fatal("artifact SHA256 is empty")
	}
	if len(document.Components) == 0 {
		t.Fatal("components empty, want Go module evidence")
	}
}

func TestGenerateReleaseNotesIncludesReleaseEvidenceAndLimitations(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	output := filepath.Join(root, "RELEASE_NOTES.md")

	result, err := GenerateReleaseNotes(ReleaseNotesOptions{
		Version: "v0.1.0",
		Output:  output,
	})
	if err != nil {
		t.Fatalf("GenerateReleaseNotes() error = %v", err)
	}
	if result.OutputPath != output {
		t.Fatalf("output = %q, want %q", result.OutputPath, output)
	}
	notes, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile() notes error = %v", err)
	}
	text := string(notes)
	for _, snippet := range []string{
		"# Cachy v0.1.0 Release Notes",
		"## Artifacts",
		"checksums",
		"SBOM",
		"Docker",
		"## Verification",
		"## Known Limitations",
		"Publishing public packages requires explicit owner approval.",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("release notes missing %q", snippet)
		}
	}
}
