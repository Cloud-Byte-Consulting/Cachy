package plugin

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	SupportedAPIVersion = "v1"

	MinTimeoutMS = 1
	MaxTimeoutMS = 1000

	MinMemoryMB = 1
	MaxMemoryMB = 128

	MinInputBytes = 1
	MaxInputBytes = 1 << 20

	MinOutputBytes = 1
	MaxOutputBytes = 512 << 10
)

var (
	ErrInvalidManifest = errors.New("invalid plugin manifest")
	manifestNameRE     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,62}[a-z0-9]$`)
	versionRE          = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

	supportedCompressCapabilities = map[string]struct{}{
		"code": {},
		"diff": {},
		"json": {},
		"log":  {},
		"text": {},
	}
)

type Manifest struct {
	Name         string       `toml:"name"`
	Version      string       `toml:"version"`
	APIVersion   string       `toml:"api_version"`
	Description  string       `toml:"description"`
	Capabilities Capabilities `toml:"capabilities"`
	Limits       Limits       `toml:"limits"`
}

type Capabilities struct {
	Compress []string `toml:"compress"`
	Classify []string `toml:"classify"`
	Redact   []string `toml:"redact"`
}

type Limits struct {
	TimeoutMS      int `toml:"timeout_ms"`
	MemoryMB       int `toml:"memory_mb"`
	MaxInputBytes  int `toml:"max_input_bytes"`
	MaxOutputBytes int `toml:"max_output_bytes"`
}

func LoadManifest(path string) (Manifest, error) {
	var manifest Manifest
	if _, err := toml.DecodeFile(path, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("%w: parse %s: %v", ErrInvalidManifest, path, err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func ValidateManifest(manifest Manifest) error {
	var problems []string

	if strings.TrimSpace(manifest.Name) == "" {
		problems = append(problems, "name is required")
	} else if !manifestNameRE.MatchString(manifest.Name) {
		problems = append(problems, "name must use lowercase letters, numbers, dots, underscores, or dashes")
	}

	if strings.TrimSpace(manifest.Version) == "" {
		problems = append(problems, "version is required")
	} else if !versionRE.MatchString(manifest.Version) {
		problems = append(problems, "version must use semantic version form")
	}

	if manifest.APIVersion != SupportedAPIVersion {
		problems = append(problems, fmt.Sprintf("unsupported api_version %q", manifest.APIVersion))
	}

	if len(manifest.Capabilities.Compress) == 0 && len(manifest.Capabilities.Classify) == 0 && len(manifest.Capabilities.Redact) == 0 {
		problems = append(problems, "at least one supported capability is required")
	}
	for _, capability := range manifest.Capabilities.Compress {
		if _, ok := supportedCompressCapabilities[capability]; !ok {
			problems = append(problems, fmt.Sprintf("unsupported compress capability %q", capability))
		}
	}
	if len(manifest.Capabilities.Classify) > 0 {
		problems = append(problems, "classify capabilities are not supported yet")
	}
	if len(manifest.Capabilities.Redact) > 0 {
		problems = append(problems, "redact capabilities are not supported yet")
	}

	problems = appendLimitProblem(problems, "timeout_ms", manifest.Limits.TimeoutMS, MinTimeoutMS, MaxTimeoutMS)
	problems = appendLimitProblem(problems, "memory_mb", manifest.Limits.MemoryMB, MinMemoryMB, MaxMemoryMB)
	problems = appendLimitProblem(problems, "max_input_bytes", manifest.Limits.MaxInputBytes, MinInputBytes, MaxInputBytes)
	problems = appendLimitProblem(problems, "max_output_bytes", manifest.Limits.MaxOutputBytes, MinOutputBytes, MaxOutputBytes)
	if manifest.Limits.MaxOutputBytes > 0 && manifest.Limits.MaxInputBytes > 0 && manifest.Limits.MaxOutputBytes > manifest.Limits.MaxInputBytes {
		problems = append(problems, "max_output_bytes must be less than or equal to max_input_bytes")
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidManifest, strings.Join(problems, "; "))
	}
	return nil
}

func appendLimitProblem(problems []string, name string, value, min, max int) []string {
	if value < min || value > max {
		return append(problems, fmt.Sprintf("%s must be between %d and %d", name, min, max))
	}
	return problems
}
