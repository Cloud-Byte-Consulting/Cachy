package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexInstallDryRunShowsChangesWithoutWriting(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	original := "model = \"gpt-5\"\n"
	if err := os.WriteFile(configPath, []byte(original), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := ApplyCodexIntegration(CodexOptions{
		Action:       CodexInstall,
		ConfigPath:   configPath,
		ProxyBaseURL: "http://127.0.0.1:8787/v1",
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("ApplyCodexIntegration() error = %v", err)
	}

	if !result.DryRun || !result.Changed {
		t.Fatalf("result = %#v, want dry-run changed plan", result)
	}
	got := readFile(t, configPath)
	if got != original {
		t.Fatalf("dry run wrote config: %q", got)
	}
	if !strings.Contains(result.Preview, ManagedCodexBegin) || !strings.Contains(result.Preview, "http://127.0.0.1:8787/v1") {
		t.Fatalf("preview missing managed block: %q", result.Preview)
	}
}

func TestCodexInstallAndUninstallOnlyTouchManagedBlock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	original := "model = \"gpt-5\"\n\n[mcp_servers.existing]\ncommand = \"tool\"\n"
	if err := os.WriteFile(configPath, []byte(original), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	installed, err := ApplyCodexIntegration(CodexOptions{
		Action:       CodexInstall,
		ConfigPath:   configPath,
		ProxyBaseURL: "http://127.0.0.1:8787/v1",
	})
	if err != nil {
		t.Fatalf("install error = %v", err)
	}
	if !installed.Changed {
		t.Fatal("install changed = false, want true")
	}
	afterInstall := readFile(t, configPath)
	if !strings.Contains(afterInstall, original) || !strings.Contains(afterInstall, ManagedCodexBegin) {
		t.Fatalf("installed config did not preserve original plus managed block:\n%s", afterInstall)
	}

	uninstalled, err := ApplyCodexIntegration(CodexOptions{
		Action:     CodexUninstall,
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("uninstall error = %v", err)
	}
	if !uninstalled.Changed {
		t.Fatal("uninstall changed = false, want true")
	}
	if got := readFile(t, configPath); got != original {
		t.Fatalf("uninstall config = %q, want original %q", got, original)
	}
}

func TestCodexRepairReplacesStaleManagedBlock(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	stale := "model = \"gpt-5\"\n\n" + ManagedCodexBegin + "\nold = true\n" + ManagedCodexEnd + "\n"
	if err := os.WriteFile(configPath, []byte(stale), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := ApplyCodexIntegration(CodexOptions{
		Action:       CodexRepair,
		ConfigPath:   configPath,
		ProxyBaseURL: "http://127.0.0.1:8787/v1",
	})
	if err != nil {
		t.Fatalf("repair error = %v", err)
	}
	if !result.Changed {
		t.Fatal("repair changed = false, want true")
	}

	got := readFile(t, configPath)
	if strings.Contains(got, "old = true") {
		t.Fatalf("repair left stale managed content:\n%s", got)
	}
	if !strings.Contains(got, `base_url = "http://127.0.0.1:8787/v1"`) {
		t.Fatalf("repair missing new proxy URL:\n%s", got)
	}
}

func TestCodexInstallDoesNotOverrideExistingProviderSelection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	if err := os.WriteFile(configPath, []byte("model_provider = \"openai\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := ApplyCodexIntegration(CodexOptions{
		Action:       CodexInstall,
		ConfigPath:   configPath,
		ProxyBaseURL: "http://127.0.0.1:8787/v1",
	})
	if err != nil {
		t.Fatalf("install error = %v", err)
	}

	got := readFile(t, configPath)
	if strings.Count(got, "model_provider =") != 1 {
		t.Fatalf("config has duplicate model_provider values:\n%s", got)
	}
	if !strings.Contains(result.Message, "existing model_provider") {
		t.Fatalf("message = %q, want existing provider warning", result.Message)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}
