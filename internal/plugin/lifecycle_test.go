package plugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListPluginsReportsEnabledState(t *testing.T) {
	root := t.TempDir()
	writePluginManifest(t, root, "stacktrace-compressor")
	writePluginManifest(t, root, "json-compressor")
	if err := SetEnabled(root, "json-compressor", true); err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}

	plugins, err := ListPlugins(root)
	if err != nil {
		t.Fatalf("ListPlugins() error = %v", err)
	}

	if len(plugins) != 2 {
		t.Fatalf("len(plugins) = %d, want 2", len(plugins))
	}
	if plugins[0].Manifest.Name != "json-compressor" || !plugins[0].Enabled {
		t.Fatalf("plugins[0] = %#v, want enabled json-compressor first by name", plugins[0])
	}
	if plugins[1].Manifest.Name != "stacktrace-compressor" || plugins[1].Enabled {
		t.Fatalf("plugins[1] = %#v, want disabled stacktrace-compressor", plugins[1])
	}
}

func TestInspectPluginRejectsMissingPlugin(t *testing.T) {
	_, err := InspectPlugin(t.TempDir(), "missing")
	if !errors.Is(err, ErrPluginNotFound) {
		t.Fatalf("InspectPlugin() error = %v, want ErrPluginNotFound", err)
	}
}

func TestListPluginsRejectsInvalidManifest(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "bad-plugin")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, ManifestFileName), []byte("api_version = \"v2\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := ListPlugins(root)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("ListPlugins() error = %v, want ErrInvalidManifest", err)
	}
}

func TestEnableDisablePluginUsesMarkerFile(t *testing.T) {
	root := t.TempDir()
	writePluginManifest(t, root, "stacktrace-compressor")

	if err := SetEnabled(root, "stacktrace-compressor", true); err != nil {
		t.Fatalf("enable SetEnabled() error = %v", err)
	}
	info, err := InspectPlugin(root, "stacktrace-compressor")
	if err != nil {
		t.Fatalf("InspectPlugin() error = %v", err)
	}
	if !info.Enabled {
		t.Fatal("plugin is disabled, want enabled")
	}

	if err := SetEnabled(root, "stacktrace-compressor", false); err != nil {
		t.Fatalf("disable SetEnabled() error = %v", err)
	}
	info, err = InspectPlugin(root, "stacktrace-compressor")
	if err != nil {
		t.Fatalf("InspectPlugin() after disable error = %v", err)
	}
	if info.Enabled {
		t.Fatal("plugin is enabled, want disabled")
	}
}

func TestTestPluginRejectsDisabledPlugin(t *testing.T) {
	root := t.TempDir()
	writePluginManifest(t, root, "stacktrace-compressor")
	fixture := filepath.Join(root, "fixture.log")
	if err := os.WriteFile(fixture, []byte("line one\nline two\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() fixture error = %v", err)
	}

	_, err := TestPlugin(context.Background(), TestOptions{
		RootDir:     root,
		Name:        "stacktrace-compressor",
		FixturePath: fixture,
		Runner: func(context.Context, Manifest, []byte, RunInput) (PluginOutput, error) {
			t.Fatal("runner should not be called for disabled plugin")
			return PluginOutput{}, nil
		},
	})
	if !errors.Is(err, ErrPluginDisabled) {
		t.Fatalf("TestPlugin() error = %v, want ErrPluginDisabled", err)
	}
}

func TestTestPluginRunsFixtureThroughHostRunner(t *testing.T) {
	root := t.TempDir()
	writePluginManifest(t, root, "stacktrace-compressor")
	writePluginWASM(t, root, "stacktrace-compressor", []byte("wasm-bytes"))
	if err := SetEnabled(root, "stacktrace-compressor", true); err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}
	fixture := filepath.Join(root, "fixture.log")
	if err := os.WriteFile(fixture, []byte("line one\nline two\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() fixture error = %v", err)
	}

	result, err := TestPlugin(context.Background(), TestOptions{
		RootDir:     root,
		Name:        "stacktrace-compressor",
		FixturePath: fixture,
		Runner: func(_ context.Context, manifest Manifest, wasm []byte, input RunInput) (PluginOutput, error) {
			if manifest.Name != "stacktrace-compressor" {
				t.Fatalf("manifest.Name = %q, want stacktrace-compressor", manifest.Name)
			}
			if string(wasm) != "wasm-bytes" {
				t.Fatalf("wasm = %q, want wasm-bytes", string(wasm))
			}
			if input.Block.Text != "line one\nline two\n" || input.Block.ID != "fixture" {
				t.Fatalf("input.Block = %#v, want fixture text", input.Block)
			}
			return PluginOutput{Action: "replace", Text: "short"}, nil
		},
	})
	if err != nil {
		t.Fatalf("TestPlugin() error = %v", err)
	}
	if result.Output.Action != "replace" || result.Output.Text != "short" {
		t.Fatalf("TestPlugin() result = %#v, want replace short", result)
	}
}

func writePluginManifest(t *testing.T, root, name string) {
	t.Helper()

	pluginDir := filepath.Join(root, name)
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := strings.ReplaceAll(validManifestTOML, "stacktrace-compressor", name)
	if err := os.WriteFile(filepath.Join(pluginDir, ManifestFileName), []byte(manifest), 0o600); err != nil {
		t.Fatalf("WriteFile() manifest error = %v", err)
	}
}

func writePluginWASM(t *testing.T, root, name string, wasm []byte) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(root, name, WASMFileName), wasm, 0o600); err != nil {
		t.Fatalf("WriteFile() wasm error = %v", err)
	}
}

const validManifestTOML = `
name = "stacktrace-compressor"
version = "0.1.0"
api_version = "v1"
description = "Compresses repeated stack frames and dependency noise."

[capabilities]
compress = ["log", "text"]
classify = []
redact = []

[limits]
timeout_ms = 50
memory_mb = 32
max_input_bytes = 262144
max_output_bytes = 131072
`
