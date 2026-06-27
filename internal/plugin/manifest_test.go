package plugin

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifestAcceptsValidFixture(t *testing.T) {
	manifest, err := LoadManifest(filepath.Join("testdata", "valid-compressor.toml"))
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}

	if manifest.Name != "stacktrace-compressor" {
		t.Fatalf("Name = %q, want stacktrace-compressor", manifest.Name)
	}
	if manifest.APIVersion != "v1" {
		t.Fatalf("APIVersion = %q, want v1", manifest.APIVersion)
	}
	if got := manifest.Capabilities.Compress; len(got) != 2 || got[0] != "log" || got[1] != "text" {
		t.Fatalf("Capabilities.Compress = %#v, want [log text]", got)
	}
	if manifest.Limits.TimeoutMS != 50 {
		t.Fatalf("Limits.TimeoutMS = %d, want 50", manifest.Limits.TimeoutMS)
	}
}

func TestValidateManifestRejectsMissingRequiredFields(t *testing.T) {
	manifest := Manifest{
		Version:    "0.1.0",
		APIVersion: "v1",
		Limits: Limits{
			TimeoutMS:      50,
			MemoryMB:       32,
			MaxInputBytes:  262144,
			MaxOutputBytes: 131072,
		},
	}

	err := ValidateManifest(manifest)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("ValidateManifest error = %v, want ErrInvalidManifest", err)
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("ValidateManifest error = %v, want missing name detail", err)
	}
}

func TestLoadManifestRejectsUnsupportedAPIVersion(t *testing.T) {
	_, err := LoadManifest(filepath.Join("testdata", "bad-api-version.toml"))
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("LoadManifest error = %v, want ErrInvalidManifest", err)
	}
	if !strings.Contains(err.Error(), "unsupported api_version") {
		t.Fatalf("LoadManifest error = %v, want api_version detail", err)
	}
}

func TestLoadManifestRejectsUnsupportedCapabilities(t *testing.T) {
	_, err := LoadManifest(filepath.Join("testdata", "bad-capability.toml"))
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("LoadManifest error = %v, want ErrInvalidManifest", err)
	}
	if !strings.Contains(err.Error(), "unsupported compress capability") {
		t.Fatalf("LoadManifest error = %v, want capability detail", err)
	}
}

func TestLoadManifestRejectsUnsafeLimits(t *testing.T) {
	_, err := LoadManifest(filepath.Join("testdata", "unsafe-limits.toml"))
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("LoadManifest error = %v, want ErrInvalidManifest", err)
	}
	for _, want := range []string{"timeout_ms must be between", "memory_mb must be between", "max_output_bytes must be between"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("LoadManifest error = %v, want detail %q", err, want)
		}
	}
}

func TestValidateManifestRejectsNonEmptyDeferredCapabilities(t *testing.T) {
	manifest := validManifest()
	manifest.Capabilities.Redact = []string{"pii"}

	err := ValidateManifest(manifest)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("ValidateManifest error = %v, want ErrInvalidManifest", err)
	}
	if !strings.Contains(err.Error(), "redact capabilities are not supported yet") {
		t.Fatalf("ValidateManifest error = %v, want redaction detail", err)
	}
}

func validManifest() Manifest {
	return Manifest{
		Name:        "stacktrace-compressor",
		Version:     "0.1.0",
		APIVersion:  "v1",
		Description: "Compresses repeated stack frames.",
		Capabilities: Capabilities{
			Compress: []string{"log", "text"},
		},
		Limits: Limits{
			TimeoutMS:      50,
			MemoryMB:       32,
			MaxInputBytes:  262144,
			MaxOutputBytes: 131072,
		},
	}
}
