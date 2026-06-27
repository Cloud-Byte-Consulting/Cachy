package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

type SBOMOptions struct {
	Kind        string
	Artifact    string
	OutputPath  string
	Version     string
	GeneratedAt time.Time
}

type SBOMResult struct {
	OutputPath string
}

type SBOMDocument struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Kind        string          `json:"kind"`
	GeneratedAt string          `json:"generated_at"`
	Artifact    SBOMArtifact    `json:"artifact"`
	Components  []SBOMComponent `json:"components"`
}

type SBOMArtifact struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256,omitempty"`
}

type SBOMComponent struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type ReleaseNotesOptions struct {
	Version string
	Output  string
}

type ReleaseNotesResult struct {
	OutputPath string
}

func GenerateSBOM(options SBOMOptions) (SBOMResult, error) {
	if options.Kind == "" {
		return SBOMResult{}, errors.New("sbom kind is required")
	}
	if options.Artifact == "" {
		return SBOMResult{}, errors.New("artifact path is required")
	}
	if options.OutputPath == "" {
		return SBOMResult{}, errors.New("output path is required")
	}
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	artifact, err := artifactEvidence(options.Artifact)
	if err != nil {
		return SBOMResult{}, err
	}
	document := SBOMDocument{
		Name:        "cachy",
		Version:     options.Version,
		Kind:        options.Kind,
		GeneratedAt: generatedAt.Format(time.RFC3339),
		Artifact:    artifact,
		Components:  buildComponents(),
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return SBOMResult{}, err
	}
	payload = append(payload, '\n')

	if err := os.MkdirAll(filepath.Dir(options.OutputPath), 0o755); err != nil {
		return SBOMResult{}, err
	}
	if err := os.WriteFile(options.OutputPath, payload, 0o644); err != nil {
		return SBOMResult{}, err
	}
	return SBOMResult{OutputPath: options.OutputPath}, nil
}

func GenerateReleaseNotes(options ReleaseNotesOptions) (ReleaseNotesResult, error) {
	if options.Version == "" {
		return ReleaseNotesResult{}, errors.New("version is required")
	}
	if options.Output == "" {
		return ReleaseNotesResult{}, errors.New("output path is required")
	}
	notes := fmt.Sprintf(`# Cachy %s Release Notes

## Artifacts

- Windows archives are published as ZIP files with matching checksums.
- macOS and Linux archives are published as tar.gz files with matching checksums.
- Docker packaging produces a non-published multi-arch OCI artifact for linux/amd64 and linux/arm64.
- SBOM JSON files are generated for release evidence where practical.

## Verification

- Verify each archive with its .sha256 sidecar before installation.
- Inspect SBOM files alongside release artifacts.
- Run `+"`cachy doctor`"+` after placing the binary on PATH.
- Run `+"`cachy proxy --listen 127.0.0.1:8787 --target <provider-url>`"+` for a local smoke check.

## Known Limitations

- Publishing public packages requires explicit owner approval.
- Registry push, signing, notarization, Winget, Scoop, Homebrew, deb, and rpm packaging are later work.
- Docker image publication is intentionally separated from non-published OCI artifact generation.
`, options.Version)

	if err := os.MkdirAll(filepath.Dir(options.Output), 0o755); err != nil {
		return ReleaseNotesResult{}, err
	}
	if err := os.WriteFile(options.Output, []byte(notes), 0o644); err != nil {
		return ReleaseNotesResult{}, err
	}
	return ReleaseNotesResult{OutputPath: options.Output}, nil
}

func artifactEvidence(path string) (SBOMArtifact, error) {
	info, err := os.Stat(path)
	if err != nil {
		return SBOMArtifact{}, err
	}
	artifact := SBOMArtifact{
		Name:      filepath.Base(path),
		Path:      filepath.ToSlash(path),
		SizeBytes: info.Size(),
	}
	if info.IsDir() {
		return artifact, nil
	}
	sum, err := fileSHA256(path)
	if err != nil {
		return SBOMArtifact{}, err
	}
	artifact.SHA256 = sum
	return artifact, nil
}

func fileSHA256(path string) (string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func buildComponents() []SBOMComponent {
	info, ok := debug.ReadBuildInfo()
	components := []SBOMComponent{{Type: "go-module", Name: "cachy"}}
	if !ok {
		return components
	}
	if info.Main.Path != "" {
		components[0].Name = info.Main.Path
		components[0].Version = info.Main.Version
	}
	for _, dep := range info.Deps {
		version := dep.Version
		if dep.Replace != nil {
			version = dep.Replace.Version
		}
		components = append(components, SBOMComponent{
			Type:    "go-module",
			Name:    dep.Path,
			Version: version,
		})
	}
	return components
}
