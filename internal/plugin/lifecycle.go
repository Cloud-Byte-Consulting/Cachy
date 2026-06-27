package plugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	ManifestFileName = "manifest.toml"
	WASMFileName     = "plugin.wasm"
	EnabledFileName  = "enabled"
)

var (
	ErrPluginNotFound = errors.New("plugin not found")
	ErrPluginDisabled = errors.New("plugin disabled")
)

type Info struct {
	Manifest Manifest
	Dir      string
	Enabled  bool
}

type HostRunner func(context.Context, Manifest, []byte, RunInput) (PluginOutput, error)

type TestOptions struct {
	RootDir     string
	Name        string
	FixturePath string
	Runner      HostRunner
}

type TestResult struct {
	Plugin Info
	Output PluginOutput
}

func ListPlugins(root string) ([]Info, error) {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	plugins := make([]Info, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := inspectPluginDir(filepath.Join(root, entry.Name()))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, info)
	}
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Manifest.Name < plugins[j].Manifest.Name
	})
	return plugins, nil
}

func InspectPlugin(root, name string) (Info, error) {
	if !manifestNameRE.MatchString(name) {
		return Info{}, ErrPluginNotFound
	}
	info, err := inspectPluginDir(filepath.Join(root, name))
	if errors.Is(err, os.ErrNotExist) {
		return Info{}, ErrPluginNotFound
	}
	if err != nil {
		return Info{}, err
	}
	if info.Manifest.Name != name {
		return Info{}, fmt.Errorf("%w: manifest name %q does not match directory %q", ErrInvalidManifest, info.Manifest.Name, name)
	}
	return info, nil
}

func SetEnabled(root, name string, enabled bool) error {
	info, err := InspectPlugin(root, name)
	if err != nil {
		return err
	}
	marker := filepath.Join(info.Dir, EnabledFileName)
	if enabled {
		return os.WriteFile(marker, []byte("enabled\n"), 0o600)
	}
	err = os.Remove(marker)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func TestPlugin(ctx context.Context, options TestOptions) (TestResult, error) {
	info, err := InspectPlugin(options.RootDir, options.Name)
	if err != nil {
		return TestResult{}, err
	}
	if !info.Enabled {
		return TestResult{}, ErrPluginDisabled
	}
	fixture, err := os.ReadFile(options.FixturePath)
	if err != nil {
		return TestResult{}, err
	}
	wasm, err := os.ReadFile(filepath.Join(info.Dir, WASMFileName))
	if err != nil {
		return TestResult{}, err
	}
	runner := options.Runner
	if runner == nil {
		runner = RunWASM
	}
	output, err := runner(ctx, info.Manifest, wasm, RunInput{
		RequestID:   "plugin-test",
		Provider:    "fixture",
		Model:       "fixture",
		ContentType: firstCompressCapability(info.Manifest),
		Block: BlockInput{
			ID:   "fixture",
			Role: "tool",
			Text: string(fixture),
		},
		Metadata: map[string]any{"source": "fixture"},
	})
	if err != nil {
		return TestResult{}, err
	}
	return TestResult{Plugin: info, Output: output}, nil
}

func inspectPluginDir(dir string) (Info, error) {
	manifestPath := filepath.Join(dir, ManifestFileName)
	if _, err := os.Stat(manifestPath); err != nil {
		return Info{}, err
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return Info{}, err
	}
	_, err = os.Stat(filepath.Join(dir, EnabledFileName))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Info{}, err
	}
	return Info{
		Manifest: manifest,
		Dir:      dir,
		Enabled:  err == nil,
	}, nil
}

func firstCompressCapability(manifest Manifest) string {
	if len(manifest.Capabilities.Compress) == 0 {
		return "text"
	}
	return manifest.Capabilities.Compress[0]
}
