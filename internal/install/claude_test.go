package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeInstallDryRunShowsSettingsWithoutWriting(t *testing.T) {
	t.Parallel()

	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	result, err := ApplyClaudeIntegration(ClaudeOptions{
		Action:           ClaudeInstall,
		SettingsPath:     settingsPath,
		AnthropicBaseURL: "http://127.0.0.1:8787",
		DryRun:           true,
	})
	if err != nil {
		t.Fatalf("ApplyClaudeIntegration() error = %v", err)
	}

	if !result.DryRun || !result.Changed {
		t.Fatalf("result = %#v, want dry-run changed plan", result)
	}
	if _, err := os.Stat(settingsPath); !os.IsNotExist(err) {
		t.Fatalf("dry run stat error = %v, want not exist", err)
	}
	if !strings.Contains(result.Preview, "ANTHROPIC_BASE_URL") || !strings.Contains(result.Preview, "http://127.0.0.1:8787") {
		t.Fatalf("preview missing Claude env setup: %q", result.Preview)
	}
}

func TestClaudeInstallAndUninstallOnlyTouchCachySettings(t *testing.T) {
	t.Parallel()

	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	original := `{"permissions":{"defaultMode":"auto"},"effortLevel":"medium"}`
	if err := os.WriteFile(settingsPath, []byte(original), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	installed, err := ApplyClaudeIntegration(ClaudeOptions{
		Action:           ClaudeInstall,
		SettingsPath:     settingsPath,
		AnthropicBaseURL: "http://127.0.0.1:8787",
	})
	if err != nil {
		t.Fatalf("install error = %v", err)
	}
	if !installed.Changed {
		t.Fatal("install changed = false, want true")
	}
	afterInstall := readJSONMap(t, settingsPath)
	if afterInstall["permissions"] == nil || afterInstall["effortLevel"] != "medium" {
		t.Fatalf("install lost unrelated settings: %#v", afterInstall)
	}
	env := afterInstall["env"].(map[string]any)
	if env["ANTHROPIC_BASE_URL"] != "http://127.0.0.1:8787" {
		t.Fatalf("ANTHROPIC_BASE_URL = %#v, want Cachy URL", env["ANTHROPIC_BASE_URL"])
	}

	uninstalled, err := ApplyClaudeIntegration(ClaudeOptions{
		Action:       ClaudeUninstall,
		SettingsPath: settingsPath,
	})
	if err != nil {
		t.Fatalf("uninstall error = %v", err)
	}
	if !uninstalled.Changed {
		t.Fatal("uninstall changed = false, want true")
	}
	afterUninstall := readJSONMap(t, settingsPath)
	if _, ok := afterUninstall["cachy"]; ok {
		t.Fatalf("uninstall left cachy metadata: %#v", afterUninstall)
	}
	if env, ok := afterUninstall["env"].(map[string]any); ok {
		if _, exists := env["ANTHROPIC_BASE_URL"]; exists {
			t.Fatalf("uninstall left managed ANTHROPIC_BASE_URL: %#v", env)
		}
	}
	if afterUninstall["permissions"] == nil || afterUninstall["effortLevel"] != "medium" {
		t.Fatalf("uninstall lost unrelated settings: %#v", afterUninstall)
	}
}

func TestClaudeRepairReplacesStaleManagedSettings(t *testing.T) {
	t.Parallel()

	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	stale := `{"env":{"ANTHROPIC_BASE_URL":"http://old.local"},"cachy":{"managed":true,"anthropic_base_url":"http://old.local"}}`
	if err := os.WriteFile(settingsPath, []byte(stale), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := ApplyClaudeIntegration(ClaudeOptions{
		Action:           ClaudeRepair,
		SettingsPath:     settingsPath,
		AnthropicBaseURL: "http://127.0.0.1:8787",
	})
	if err != nil {
		t.Fatalf("repair error = %v", err)
	}
	if !result.Changed {
		t.Fatal("repair changed = false, want true")
	}
	settings := readJSONMap(t, settingsPath)
	env := settings["env"].(map[string]any)
	if env["ANTHROPIC_BASE_URL"] != "http://127.0.0.1:8787" {
		t.Fatalf("repair env = %#v, want new URL", env)
	}
}

func TestClaudeInstallPreservesExistingAnthropicBaseURL(t *testing.T) {
	t.Parallel()

	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	existing := `{"env":{"ANTHROPIC_BASE_URL":"http://custom.local"}}`
	if err := os.WriteFile(settingsPath, []byte(existing), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := ApplyClaudeIntegration(ClaudeOptions{
		Action:           ClaudeInstall,
		SettingsPath:     settingsPath,
		AnthropicBaseURL: "http://127.0.0.1:8787",
	})
	if err != nil {
		t.Fatalf("install error = %v", err)
	}
	settings := readJSONMap(t, settingsPath)
	env := settings["env"].(map[string]any)
	if env["ANTHROPIC_BASE_URL"] != "http://custom.local" {
		t.Fatalf("existing ANTHROPIC_BASE_URL overwritten: %#v", env)
	}
	if !strings.Contains(result.Message, "existing ANTHROPIC_BASE_URL") {
		t.Fatalf("message = %q, want existing env warning", result.Message)
	}
}

func readJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("Unmarshal() error = %v\n%s", err, data)
	}
	return value
}
