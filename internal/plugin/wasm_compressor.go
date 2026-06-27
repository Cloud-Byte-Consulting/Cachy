package plugin

import (
	"context"
	"os"
	"path/filepath"

	"github.com/cloud-byte-consulting/cachy/internal/compress"
)

type ReadWASMFunc func(Info) ([]byte, error)

type WASMCompressor struct {
	Plugins   []Info
	Runner    HostRunner
	ReadWASM  ReadWASMFunc
	RequestID string
	Model     string
}

func LoadWASMCompressor(root string) (WASMCompressor, error) {
	plugins, err := ListPlugins(root)
	if err != nil {
		return WASMCompressor{}, err
	}
	enabled := make([]Info, 0, len(plugins))
	for _, info := range plugins {
		if info.Enabled {
			enabled = append(enabled, info)
		}
	}
	return WASMCompressor{Plugins: enabled}, nil
}

func (c WASMCompressor) Compress(block compress.Block) (compress.Proposal, error) {
	runner := c.Runner
	if runner == nil {
		runner = RunWASM
	}
	readWASM := c.ReadWASM
	if readWASM == nil {
		readWASM = readPluginWASM
	}

	for _, info := range c.Plugins {
		if !canPluginCompress(info, block.ContentType) {
			continue
		}
		wasm, err := readWASM(info)
		if err != nil {
			return compress.Proposal{}, err
		}
		output, err := runner(context.Background(), info.Manifest, wasm, RunInput{
			APIVersion:  SupportedAPIVersion,
			RequestID:   c.RequestID,
			Provider:    string(block.Provider),
			Model:       c.Model,
			ContentType: string(block.ContentType),
			Block: BlockInput{
				ID:   block.ID,
				Role: block.Role,
				Text: block.Text,
			},
			Metadata: map[string]any{
				"path":      block.Path,
				"source":    string(block.Source),
				"stability": string(block.Stability),
			},
		})
		if err != nil {
			return compress.Proposal{}, err
		}
		if output.Action == "replace" {
			return compress.Proposal{Text: output.Text}, nil
		}
		return compress.Proposal{Text: block.Text}, nil
	}

	return compress.Proposal{Text: block.Text}, nil
}

func canPluginCompress(info Info, contentType compress.ContentType) bool {
	if !info.Enabled {
		return false
	}
	for _, capability := range info.Manifest.Capabilities.Compress {
		if capability == string(contentType) {
			return true
		}
	}
	return false
}

func readPluginWASM(info Info) ([]byte, error) {
	return os.ReadFile(filepath.Join(info.Dir, WASMFileName))
}
