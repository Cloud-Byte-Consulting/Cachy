package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
	"unicode/utf8"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var ErrPluginExecution = errors.New("plugin execution failed")

type RunInput struct {
	APIVersion  string         `json:"api_version"`
	RequestID   string         `json:"request_id"`
	Provider    string         `json:"provider"`
	Model       string         `json:"model"`
	ContentType string         `json:"content_type"`
	Block       BlockInput     `json:"block"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type BlockInput struct {
	ID   string `json:"id"`
	Role string `json:"role"`
	Text string `json:"text"`
}

type PluginOutput struct {
	Action     string  `json:"action"`
	Text       string  `json:"text,omitempty"`
	Summary    string  `json:"summary,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Lossiness  string  `json:"lossiness,omitempty"`
}

func RunWASM(ctx context.Context, manifest Manifest, wasm []byte, input RunInput) (PluginOutput, error) {
	if err := ValidateManifest(manifest); err != nil {
		return PluginOutput{}, err
	}
	if input.APIVersion == "" {
		input.APIVersion = SupportedAPIVersion
	}

	inputPayload, err := json.Marshal(input)
	if err != nil {
		return PluginOutput{}, fmt.Errorf("%w: encode selected block input: %v", ErrPluginExecution, err)
	}
	if len(inputPayload) > manifest.Limits.MaxInputBytes {
		return PluginOutput{}, fmt.Errorf("%w: selected block input exceeds max_input_bytes", ErrPluginExecution)
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(manifest.Limits.TimeoutMS)*time.Millisecond)
	defer cancel()

	runtimeConfig := wazero.NewRuntimeConfigInterpreter().
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(memoryPages(manifest.Limits.MemoryMB))
	runtime := wazero.NewRuntimeWithConfig(runCtx, runtimeConfig)
	defer runtime.Close(context.Background())

	var output bytes.Buffer
	hostBuilder := runtime.NewHostModuleBuilder("cachy")
	hostBuilder.NewFunctionBuilder().WithFunc(func() uint32 {
		return uint32(len(inputPayload))
	}).Export("input_len")
	hostBuilder.NewFunctionBuilder().WithFunc(func(ctx context.Context, module api.Module, ptr uint32) uint32 {
		if !module.Memory().Write(ptr, inputPayload) {
			return 0
		}
		return uint32(len(inputPayload))
	}).Export("input_read")
	hostBuilder.NewFunctionBuilder().WithFunc(func(ctx context.Context, module api.Module, ptr, byteCount uint32) uint32 {
		if byteCount > uint32(manifest.Limits.MaxOutputBytes) {
			return 0
		}
		payload, ok := module.Memory().Read(ptr, byteCount)
		if !ok {
			return 0
		}
		output.Reset()
		output.Write(payload)
		return byteCount
	}).Export("output_write")
	if _, err := hostBuilder.Instantiate(runCtx); err != nil {
		return PluginOutput{}, fmt.Errorf("%w: instantiate host imports: %v", ErrPluginExecution, err)
	}

	module, err := runtime.Instantiate(runCtx, wasm)
	if err != nil {
		return PluginOutput{}, fmt.Errorf("%w: instantiate module: %v", ErrPluginExecution, err)
	}
	defer module.Close(context.Background())

	run := module.ExportedFunction("run")
	if run == nil {
		return PluginOutput{}, fmt.Errorf("%w: exported run function is required", ErrPluginExecution)
	}
	if _, err := run.Call(runCtx); err != nil {
		return PluginOutput{}, fmt.Errorf("%w: run plugin: %w", ErrPluginExecution, err)
	}
	if output.Len() == 0 {
		return PluginOutput{}, fmt.Errorf("%w: plugin wrote no output or exceeded max_output_bytes", ErrPluginExecution)
	}
	if output.Len() > manifest.Limits.MaxOutputBytes {
		return PluginOutput{}, fmt.Errorf("%w: plugin output exceeds max_output_bytes", ErrPluginExecution)
	}
	if !utf8.Valid(output.Bytes()) {
		return PluginOutput{}, fmt.Errorf("%w: decode plugin output: invalid UTF-8", ErrPluginExecution)
	}

	var pluginOutput PluginOutput
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pluginOutput); err != nil {
		return PluginOutput{}, fmt.Errorf("%w: decode plugin output: %v", ErrPluginExecution, err)
	}
	if err := validatePluginOutput(pluginOutput); err != nil {
		return PluginOutput{}, err
	}
	return pluginOutput, nil
}

func memoryPages(memoryMB int) uint32 {
	bytes := float64(memoryMB * 1024 * 1024)
	return uint32(math.Ceil(bytes / 65536))
}

func validatePluginOutput(output PluginOutput) error {
	switch output.Action {
	case "keep", "replace", "annotate", "reject":
		return nil
	default:
		return fmt.Errorf("%w: unsupported plugin action %q", ErrPluginExecution, output.Action)
	}
}
