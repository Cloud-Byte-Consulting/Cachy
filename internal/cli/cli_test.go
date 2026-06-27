package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloud-byte-consulting/cachy/internal/doctor"
	"github.com/cloud-byte-consulting/cachy/internal/plugin"
)

func TestRunProxyRequiresTarget(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	code := Run([]string{"proxy"}, func(string) string { return "" }, nil, &stderr, func(ProxyConfig) error {
		t.Fatal("serve should not be called without target")
		return nil
	})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if got := stderr.String(); got == "" {
		t.Fatal("stderr is empty, want validation error")
	}
}

func TestRunProxyUsesDefaultsAndEnvironmentTarget(t *testing.T) {
	t.Parallel()

	var got ProxyConfig
	code := Run([]string{"proxy"}, func(key string) string {
		if key == "CACHY_TARGET_BASE_URL" {
			return "http://upstream.example"
		}
		return ""
	}, nil, nil, func(config ProxyConfig) error {
		got = config
		return nil
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got.Listen != DefaultListenAddress {
		t.Fatalf("listen = %q, want %q", got.Listen, DefaultListenAddress)
	}
	if got.TargetBaseURL != "http://upstream.example" {
		t.Fatalf("target = %q, want env target", got.TargetBaseURL)
	}
}

func TestRunProxyUsesExplicitFlags(t *testing.T) {
	t.Parallel()

	var got ProxyConfig
	code := Run([]string{"proxy", "--listen", "127.0.0.1:9999", "--target", "http://target.example"}, func(string) string {
		return ""
	}, nil, nil, func(config ProxyConfig) error {
		got = config
		return nil
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got.Listen != "127.0.0.1:9999" {
		t.Fatalf("listen = %q, want explicit listen", got.Listen)
	}
	if got.TargetBaseURL != "http://target.example" {
		t.Fatalf("target = %q, want explicit target", got.TargetBaseURL)
	}
}

func TestRunProxyReturnsOneWhenServeFails(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	code := Run([]string{"proxy", "--target", "http://target.example"}, func(string) string {
		return ""
	}, nil, &stderr, func(ProxyConfig) error {
		return errors.New("listen failed")
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if got := stderr.String(); got == "" {
		t.Fatal("stderr is empty, want serve error")
	}
}

func TestRunDoctorPrintsReportWithoutSecrets(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	code := RunWithDeps([]string{"doctor", "--target", "http://target.example"}, Deps{
		Getenv: func(key string) string {
			switch key {
			case "OPENAI_API_KEY":
				return "secret-key"
			case "HOME":
				return "/home/alex"
			case "CODEX_HOME":
				return "/home/alex/codex"
			case "CLAUDE_HOME":
				return "/home/alex/claude"
			default:
				return ""
			}
		},
		Stdout: &stdout,
		RunDoctor: func(config doctor.Config) doctor.Report {
			if config.TargetURL != "http://target.example" {
				t.Fatalf("doctor target = %q, want explicit target", config.TargetURL)
			}
			if config.CodexConfigPath != filepath.Join("/home/alex/codex", "config.toml") {
				t.Fatalf("doctor Codex path = %q, want CODEX_HOME config", config.CodexConfigPath)
			}
			if config.ClaudeSettingsPath != filepath.Join("/home/alex/claude", "settings.json") {
				t.Fatalf("doctor Claude path = %q, want CLAUDE_HOME settings", config.ClaudeSettingsPath)
			}
			return doctor.Report{Checks: []doctor.Check{
				{Name: "target-url", Status: doctor.StatusOK, Message: config.TargetURL},
				{Name: "credentials", Status: doctor.StatusOK, Message: "OPENAI_API_KEY is set"},
			}}
		},
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "OK target-url") {
		t.Fatalf("doctor output = %q, want target check", output)
	}
	if strings.Contains(output, "secret-key") {
		t.Fatalf("doctor output leaked secret: %q", output)
	}
}

func TestRunDoctorReturnsOneForFailures(t *testing.T) {
	t.Parallel()

	code := RunWithDeps([]string{"doctor"}, Deps{
		RunDoctor: func(doctor.Config) doctor.Report {
			return doctor.Report{Checks: []doctor.Check{
				{Name: "target-url", Status: doctor.StatusFail, Message: "target URL is required"},
			}}
		},
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestRunCodexIntegrationDryRunPrintsPreviewWithoutWriting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	var stdout bytes.Buffer

	code := RunWithDeps([]string{
		"integrations", "codex", "install",
		"--config", configPath,
		"--proxy-base-url", "http://127.0.0.1:8787/v1",
		"--dry-run",
	}, Deps{Stdout: &stdout})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if _, err := os.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry run config stat error = %v, want not exist", err)
	}
	if output := stdout.String(); !strings.Contains(output, "# BEGIN CACHY CODEX INTEGRATION") {
		t.Fatalf("dry-run output = %q, want managed block preview", output)
	}
}

func TestRunCodexIntegrationInstallAndUninstall(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	if err := os.WriteFile(configPath, []byte("model = \"gpt-5\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	code := RunWithDeps([]string{
		"integrations", "codex", "install",
		"--config", configPath,
		"--proxy-base-url", "http://127.0.0.1:8787/v1",
	}, Deps{})
	if code != 0 {
		t.Fatalf("install exit code = %d, want 0", code)
	}
	if got := readCLIFile(t, configPath); !strings.Contains(got, "# BEGIN CACHY CODEX INTEGRATION") {
		t.Fatalf("installed config missing managed block:\n%s", got)
	}

	code = RunWithDeps([]string{"integrations", "codex", "uninstall", "--config", configPath}, Deps{})
	if code != 0 {
		t.Fatalf("uninstall exit code = %d, want 0", code)
	}
	if got := readCLIFile(t, configPath); got != "model = \"gpt-5\"\n" {
		t.Fatalf("uninstalled config = %q, want original", got)
	}
}

func TestRunClaudeIntegrationDryRunPrintsPreviewWithoutWriting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	settingsPath := filepath.Join(root, "settings.json")
	var stdout bytes.Buffer

	code := RunWithDeps([]string{
		"integrations", "claude", "install",
		"--settings", settingsPath,
		"--anthropic-base-url", "http://127.0.0.1:8787",
		"--dry-run",
	}, Deps{Stdout: &stdout})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if _, err := os.Stat(settingsPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry run settings stat error = %v, want not exist", err)
	}
	if output := stdout.String(); !strings.Contains(output, "ANTHROPIC_BASE_URL") {
		t.Fatalf("dry-run output = %q, want Claude env preview", output)
	}
}

func TestRunClaudeIntegrationInstallAndUninstall(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	settingsPath := filepath.Join(root, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"effortLevel":"medium"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	code := RunWithDeps([]string{
		"integrations", "claude", "install",
		"--settings", settingsPath,
		"--anthropic-base-url", "http://127.0.0.1:8787",
	}, Deps{})
	if code != 0 {
		t.Fatalf("install exit code = %d, want 0", code)
	}
	if got := readCLIFile(t, settingsPath); !strings.Contains(got, "ANTHROPIC_BASE_URL") {
		t.Fatalf("installed settings missing Claude env:\n%s", got)
	}

	code = RunWithDeps([]string{"integrations", "claude", "uninstall", "--settings", settingsPath}, Deps{})
	if code != 0 {
		t.Fatalf("uninstall exit code = %d, want 0", code)
	}
	if got := readCLIFile(t, settingsPath); strings.Contains(got, "ANTHROPIC_BASE_URL") || strings.Contains(got, `"cachy"`) {
		t.Fatalf("uninstalled settings still contain Cachy integration:\n%s", got)
	}
}

func TestRunEndpointRecipePrintsGuidance(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	code := RunWithDeps([]string{
		"integrations", "recipe", "ollama",
		"--cachy-base-url", "http://127.0.0.1:8787",
	}, Deps{Stdout: &stdout})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	output := stdout.String()
	for _, want := range []string{"OPENAI_BASE_URL=http://127.0.0.1:8787/v1", "CACHY_TARGET_BASE_URL=http://127.0.0.1:11434"} {
		if !strings.Contains(output, want) {
			t.Fatalf("recipe output missing %q:\n%s", want, output)
		}
	}
}

func TestRunMCPRegistrationPrintsJSON(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	code := RunWithDeps([]string{
		"integrations", "mcp",
		"--target", "http://127.0.0.1:11434",
		"--listen", "127.0.0.1:8787",
	}, Deps{Stdout: &stdout})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	output := stdout.String()
	for _, want := range []string{`"mcpServers"`, `"command": "cachy"`, `"--target"`, `"http://127.0.0.1:11434"`} {
		if !strings.Contains(output, want) {
			t.Fatalf("MCP output missing %q:\n%s", want, output)
		}
	}
}

func TestRunPluginListAndInspect(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCLIPluginManifest(t, root, "stacktrace-compressor")
	var stdout bytes.Buffer

	code := RunWithDeps([]string{"plugin", "list", "--dir", root}, Deps{Stdout: &stdout})
	if code != 0 {
		t.Fatalf("list exit code = %d, want 0", code)
	}
	if output := stdout.String(); !strings.Contains(output, "stacktrace-compressor") || !strings.Contains(output, "disabled") {
		t.Fatalf("list output = %q, want disabled plugin", output)
	}

	stdout.Reset()
	code = RunWithDeps([]string{"plugin", "inspect", "stacktrace-compressor", "--dir", root}, Deps{Stdout: &stdout})
	if code != 0 {
		t.Fatalf("inspect exit code = %d, want 0", code)
	}
	for _, want := range []string{"name: stacktrace-compressor", "api_version: v1", "compress: log,text"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("inspect output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRunPluginEnableDisable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCLIPluginManifest(t, root, "stacktrace-compressor")

	code := RunWithDeps([]string{"plugin", "enable", "stacktrace-compressor", "--dir", root}, Deps{})
	if code != 0 {
		t.Fatalf("enable exit code = %d, want 0", code)
	}
	info, err := plugin.InspectPlugin(root, "stacktrace-compressor")
	if err != nil {
		t.Fatalf("InspectPlugin() error = %v", err)
	}
	if !info.Enabled {
		t.Fatal("plugin disabled, want enabled")
	}

	code = RunWithDeps([]string{"plugin", "disable", "stacktrace-compressor", "--dir", root}, Deps{})
	if code != 0 {
		t.Fatalf("disable exit code = %d, want 0", code)
	}
	info, err = plugin.InspectPlugin(root, "stacktrace-compressor")
	if err != nil {
		t.Fatalf("InspectPlugin() after disable error = %v", err)
	}
	if info.Enabled {
		t.Fatal("plugin enabled, want disabled")
	}
}

func TestRunPluginTestUsesFixtureRunner(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCLIPluginManifest(t, root, "stacktrace-compressor")
	writeCLIPluginWASM(t, root, "stacktrace-compressor", []byte("wasm"))
	if err := plugin.SetEnabled(root, "stacktrace-compressor", true); err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}
	fixture := filepath.Join(root, "sample.log")
	if err := os.WriteFile(fixture, []byte("hello fixture"), 0o600); err != nil {
		t.Fatalf("WriteFile() fixture error = %v", err)
	}
	var stdout bytes.Buffer

	code := RunWithDeps([]string{"plugin", "test", "stacktrace-compressor", "--dir", root, "--fixture", fixture}, Deps{
		Stdout: &stdout,
		RunPlugin: func(_ context.Context, manifest plugin.Manifest, wasm []byte, input plugin.RunInput) (plugin.PluginOutput, error) {
			if manifest.Name != "stacktrace-compressor" || string(wasm) != "wasm" || input.Block.Text != "hello fixture" {
				t.Fatalf("runner input = manifest %q wasm %q block %q", manifest.Name, string(wasm), input.Block.Text)
			}
			return plugin.PluginOutput{Action: "replace", Text: "short"}, nil
		},
	})
	if code != 0 {
		t.Fatalf("test exit code = %d, want 0", code)
	}
	if output := stdout.String(); !strings.Contains(output, "action: replace") || !strings.Contains(output, "text_bytes: 5") {
		t.Fatalf("test output = %q, want replacement summary", output)
	}
}

func TestRunPluginTestRejectsDisabledPlugin(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeCLIPluginManifest(t, root, "stacktrace-compressor")
	fixture := filepath.Join(root, "sample.log")
	if err := os.WriteFile(fixture, []byte("hello fixture"), 0o600); err != nil {
		t.Fatalf("WriteFile() fixture error = %v", err)
	}
	var stderr bytes.Buffer

	code := RunWithDeps([]string{"plugin", "test", "stacktrace-compressor", "--dir", root, "--fixture", fixture}, Deps{
		Stderr: &stderr,
		RunPlugin: func(context.Context, plugin.Manifest, []byte, plugin.RunInput) (plugin.PluginOutput, error) {
			t.Fatal("runner should not be called for disabled plugin")
			return plugin.PluginOutput{}, nil
		},
	})
	if code != 1 {
		t.Fatalf("test exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "disabled") {
		t.Fatalf("stderr = %q, want disabled detail", stderr.String())
	}
}

func readCLIFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return string(data)
}

func writeCLIPluginManifest(t *testing.T, root, name string) {
	t.Helper()

	pluginDir := filepath.Join(root, name)
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() plugin error = %v", err)
	}
	manifest := strings.ReplaceAll(validCLIManifestTOML, "stacktrace-compressor", name)
	if err := os.WriteFile(filepath.Join(pluginDir, plugin.ManifestFileName), []byte(manifest), 0o600); err != nil {
		t.Fatalf("WriteFile() manifest error = %v", err)
	}
}

func writeCLIPluginWASM(t *testing.T, root, name string, wasm []byte) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(root, name, plugin.WASMFileName), wasm, 0o600); err != nil {
		t.Fatalf("WriteFile() wasm error = %v", err)
	}
}

const validCLIManifestTOML = `
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
