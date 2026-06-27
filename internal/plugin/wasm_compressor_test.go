package plugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloud-byte-consulting/cachy/internal/compress"
)

func TestWASMCompressorMapsReplacementToCompressionProposal(t *testing.T) {
	t.Parallel()

	compressor := WASMCompressor{
		Plugins: []Info{testWASMCompressorInfo(true, "log")},
		Runner: func(_ context.Context, manifest Manifest, wasm []byte, input RunInput) (PluginOutput, error) {
			if manifest.Name != "stacktrace-compressor" {
				t.Fatalf("manifest name = %q, want stacktrace-compressor", manifest.Name)
			}
			if string(wasm) != "wasm-bytes" {
				t.Fatalf("wasm = %q, want wasm-bytes", string(wasm))
			}
			if input.APIVersion != SupportedAPIVersion || input.RequestID != "req_123" || input.Provider != "openai" || input.Model != "gpt-test" {
				t.Fatalf("input envelope = %#v, want request/provider/model metadata", input)
			}
			if input.ContentType != "log" || input.Block.ID != "block_1" || input.Block.Role != "tool" || input.Block.Text != "very long log payload" {
				t.Fatalf("input block = %#v, want selected log block", input.Block)
			}
			if input.Metadata["source"] != string(compress.SourceToolResult) || input.Metadata["path"] != "$.messages[1].content" {
				t.Fatalf("metadata = %#v, want source and path", input.Metadata)
			}
			return PluginOutput{Action: "replace", Text: "short log"}, nil
		},
		ReadWASM: func(Info) ([]byte, error) {
			return []byte("wasm-bytes"), nil
		},
		RequestID: "req_123",
		Model:     "gpt-test",
	}

	proposal, err := compressor.Compress(compress.Block{
		ID:          "block_1",
		Provider:    compress.ProviderOpenAI,
		Path:        "$.messages[1].content",
		Role:        "tool",
		Source:      compress.SourceToolResult,
		ContentType: compress.ContentLog,
		Text:        "very long log payload",
		Selected:    true,
		Stability:   compress.StabilityLive,
	})
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}
	if proposal.Text != "short log" {
		t.Fatalf("proposal text = %q, want plugin replacement", proposal.Text)
	}
}

func TestWASMCompressorParticipatesAfterNativePipelineValidation(t *testing.T) {
	t.Parallel()

	original := "plugin can reduce this repeated diagnostic payload repeated diagnostic payload repeated diagnostic payload"
	wasm := WASMCompressor{
		Plugins: []Info{testWASMCompressorInfo(true, "log")},
		Runner: func(context.Context, Manifest, []byte, RunInput) (PluginOutput, error) {
			return PluginOutput{Action: "replace", Text: "short plugin diagnostic"}, nil
		},
		ReadWASM: func(Info) ([]byte, error) {
			return []byte("wasm-bytes"), nil
		},
	}
	pipeline := compress.NewPipeline(compress.PipelineOptions{
		Compressors: []compress.Compressor{
			compress.CompressorFunc(func(compress.Block) (compress.Proposal, error) {
				return compress.Proposal{Text: original}, nil
			}),
			wasm,
		},
	})

	result, err := pipeline.Apply([]compress.Block{{
		ID:          "block_1",
		Provider:    compress.ProviderOpenAI,
		Role:        "tool",
		Source:      compress.SourceToolResult,
		ContentType: compress.ContentLog,
		Text:        original,
		Selected:    true,
		Stability:   compress.StabilityLive,
	}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if result.Blocks[0].Text != "short plugin diagnostic" {
		t.Fatalf("block text = %q, want validated WASM replacement", result.Blocks[0].Text)
	}
	if !result.Decisions[0].Applied || result.Decisions[0].Reason != compress.DecisionApplied {
		t.Fatalf("decision = %#v, want applied WASM proposal", result.Decisions[0])
	}
}

func TestWASMCompressorReplacementFallsBackWhenPipelineValidationRejectsIt(t *testing.T) {
	t.Parallel()

	original := "short original"
	wasm := WASMCompressor{
		Plugins: []Info{testWASMCompressorInfo(true, "text")},
		Runner: func(context.Context, Manifest, []byte, RunInput) (PluginOutput, error) {
			return PluginOutput{Action: "replace", Text: "larger plugin replacement with no useful token savings"}, nil
		},
		ReadWASM: func(Info) ([]byte, error) {
			return []byte("wasm-bytes"), nil
		},
	}
	pipeline := compress.NewPipeline(compress.PipelineOptions{
		Compressor: wasm,
	})

	result, err := pipeline.Apply([]compress.Block{{
		ID:          "block_1",
		Provider:    compress.ProviderOpenAI,
		Role:        "tool",
		Source:      compress.SourceToolResult,
		ContentType: compress.ContentText,
		Text:        original,
		Selected:    true,
		Stability:   compress.StabilityLive,
	}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if result.Blocks[0].Text != original {
		t.Fatalf("block text = %q, want original fallback", result.Blocks[0].Text)
	}
	if result.Decisions[0].Applied || result.Decisions[0].Reason != compress.DecisionNoSavings {
		t.Fatalf("decision = %#v, want no-savings fallback", result.Decisions[0])
	}
}

func TestWASMCompressorTreatsNonReplacementActionsAsFallback(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"keep", "annotate", "reject"} {
		t.Run(action, func(t *testing.T) {
			t.Parallel()

			compressor := WASMCompressor{
				Plugins: []Info{testWASMCompressorInfo(true, "text")},
				Runner: func(context.Context, Manifest, []byte, RunInput) (PluginOutput, error) {
					return PluginOutput{Action: action, Summary: "declined"}, nil
				},
				ReadWASM: func(Info) ([]byte, error) {
					return []byte("wasm-bytes"), nil
				},
			}

			proposal, err := compressor.Compress(compress.Block{
				ID:          "block_1",
				Provider:    compress.ProviderOpenAI,
				ContentType: compress.ContentText,
				Text:        "original block text",
				Selected:    true,
				Stability:   compress.StabilityLive,
			})
			if err != nil {
				t.Fatalf("Compress() error = %v", err)
			}
			if proposal.Text != "original block text" {
				t.Fatalf("proposal text = %q, want original fallback", proposal.Text)
			}
		})
	}
}

func TestWASMCompressorOnlyRunsEnabledMatchingPlugins(t *testing.T) {
	t.Parallel()

	compressor := WASMCompressor{
		Plugins: []Info{
			testWASMCompressorInfo(false, "log"),
			testWASMCompressorInfo(true, "json"),
		},
		Runner: func(context.Context, Manifest, []byte, RunInput) (PluginOutput, error) {
			t.Fatal("runner should not be called for disabled or capability-mismatched plugins")
			return PluginOutput{}, nil
		},
		ReadWASM: func(Info) ([]byte, error) {
			t.Fatal("WASM should not be read for disabled or capability-mismatched plugins")
			return nil, nil
		},
	}

	proposal, err := compressor.Compress(compress.Block{
		ID:          "block_1",
		Provider:    compress.ProviderOpenAI,
		ContentType: compress.ContentLog,
		Text:        "original log",
		Selected:    true,
		Stability:   compress.StabilityLive,
	})
	if err != nil {
		t.Fatalf("Compress() error = %v", err)
	}
	if proposal.Text != "original log" {
		t.Fatalf("proposal text = %q, want original fallback", proposal.Text)
	}
}

func TestWASMCompressorPropagatesPluginExecutionFailureForPipelineFallback(t *testing.T) {
	t.Parallel()

	compressor := WASMCompressor{
		Plugins: []Info{testWASMCompressorInfo(true, "log")},
		Runner: func(context.Context, Manifest, []byte, RunInput) (PluginOutput, error) {
			return PluginOutput{}, errors.New("trap")
		},
		ReadWASM: func(Info) ([]byte, error) {
			return []byte("wasm-bytes"), nil
		},
	}

	_, err := compressor.Compress(compress.Block{
		ID:          "block_1",
		Provider:    compress.ProviderOpenAI,
		ContentType: compress.ContentLog,
		Text:        "original log",
		Selected:    true,
		Stability:   compress.StabilityLive,
	})
	if err == nil {
		t.Fatal("Compress() succeeded, want plugin execution failure")
	}
}

func TestLoadWASMCompressorReadsEnabledPluginsFromDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeWASMCompressorPlugin(t, root, "stacktrace-compressor", "log")
	if err := SetEnabled(root, "stacktrace-compressor", true); err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}

	compressor, err := LoadWASMCompressor(root)
	if err != nil {
		t.Fatalf("LoadWASMCompressor() error = %v", err)
	}
	if len(compressor.Plugins) != 1 || compressor.Plugins[0].Manifest.Name != "stacktrace-compressor" {
		t.Fatalf("plugins = %#v, want enabled stacktrace-compressor", compressor.Plugins)
	}
}

func testWASMCompressorInfo(enabled bool, capabilities ...string) Info {
	return Info{
		Manifest: Manifest{
			Name:        "stacktrace-compressor",
			Version:     "0.1.0",
			APIVersion:  SupportedAPIVersion,
			Description: "Compresses logs.",
			Capabilities: Capabilities{
				Compress: capabilities,
			},
			Limits: Limits{
				TimeoutMS:      50,
				MemoryMB:       32,
				MaxInputBytes:  262144,
				MaxOutputBytes: 131072,
			},
		},
		Dir:     ".",
		Enabled: enabled,
	}
}

func writeWASMCompressorPlugin(t *testing.T, root, name, capability string) {
	t.Helper()

	pluginDir := filepath.Join(root, name)
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"" + name + "\"\n" +
		"version = \"0.1.0\"\n" +
		"api_version = \"v1\"\n" +
		"description = \"Compresses logs.\"\n\n" +
		"[capabilities]\n" +
		"compress = [\"" + capability + "\"]\n" +
		"classify = []\n" +
		"redact = []\n\n" +
		"[limits]\n" +
		"timeout_ms = 50\n" +
		"memory_mb = 32\n" +
		"max_input_bytes = 262144\n" +
		"max_output_bytes = 131072\n"
	if err := os.WriteFile(filepath.Join(pluginDir, ManifestFileName), []byte(manifest), 0o600); err != nil {
		t.Fatalf("WriteFile() manifest error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, WASMFileName), []byte("wasm-bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile() wasm error = %v", err)
	}
}
