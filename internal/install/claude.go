package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

const defaultClaudeAnthropicBaseURL = "http://127.0.0.1:8787"

type ClaudeAction string

const (
	ClaudeInstall   ClaudeAction = "install"
	ClaudeRepair    ClaudeAction = "repair"
	ClaudeUninstall ClaudeAction = "uninstall"
)

type ClaudeOptions struct {
	Action           ClaudeAction
	SettingsPath     string
	AnthropicBaseURL string
	DryRun           bool
}

type ClaudeResult struct {
	Action  ClaudeAction
	DryRun  bool
	Changed bool
	Path    string
	Preview string
	Message string
}

func DefaultClaudeSettingsPath(homeDir, claudeHome string) (string, error) {
	if claudeHome != "" {
		return filepath.Join(claudeHome, "settings.json"), nil
	}
	if homeDir == "" {
		return "", errors.New("home directory is required to resolve Claude settings")
	}
	return filepath.Join(homeDir, ".claude", "settings.json"), nil
}

func ApplyClaudeIntegration(options ClaudeOptions) (ClaudeResult, error) {
	if options.SettingsPath == "" {
		return ClaudeResult{}, errors.New("Claude settings path is required")
	}
	if options.Action == "" {
		options.Action = ClaudeInstall
	}
	if options.AnthropicBaseURL == "" {
		options.AnthropicBaseURL = defaultClaudeAnthropicBaseURL
	}
	if options.Action != ClaudeUninstall {
		if err := validateBaseURL(options.AnthropicBaseURL, "Claude Anthropic base URL"); err != nil {
			return ClaudeResult{}, err
		}
	}

	currentText, err := readOptionalFile(options.SettingsPath)
	if err != nil {
		return ClaudeResult{}, err
	}
	current, err := parseJSONMap(currentText)
	if err != nil {
		return ClaudeResult{}, err
	}
	next, message, err := planClaudeSettings(current, options)
	if err != nil {
		return ClaudeResult{}, err
	}
	nextText, err := marshalSettings(next)
	if err != nil {
		return ClaudeResult{}, err
	}
	changed := normalizeJSON(currentText) != normalizeJSON(nextText)
	result := ClaudeResult{
		Action:  options.Action,
		DryRun:  options.DryRun,
		Changed: changed,
		Path:    options.SettingsPath,
		Preview: nextText,
		Message: message,
	}
	if options.DryRun || !changed {
		return result, nil
	}
	if err := os.MkdirAll(filepath.Dir(options.SettingsPath), 0o700); err != nil {
		return ClaudeResult{}, err
	}
	if err := os.WriteFile(options.SettingsPath, []byte(nextText), 0o600); err != nil {
		return ClaudeResult{}, err
	}
	return result, nil
}

func planClaudeSettings(settings map[string]any, options ClaudeOptions) (map[string]any, string, error) {
	next := cloneJSONMap(settings)
	switch options.Action {
	case ClaudeInstall, ClaudeRepair:
		env := childMap(next, "env")
		metadata := childMap(next, "cachy")
		previousManagedURL, _ := metadata["anthropic_base_url"].(string)
		currentURL, hasCurrentURL := env["ANTHROPIC_BASE_URL"].(string)
		if hasCurrentURL && currentURL != "" && currentURL != previousManagedURL {
			metadata["managed"] = true
			metadata["anthropic_base_url"] = options.AnthropicBaseURL
			return next, "Claude Cachy metadata configured; existing ANTHROPIC_BASE_URL preserved", nil
		}
		env["ANTHROPIC_BASE_URL"] = options.AnthropicBaseURL
		metadata["managed"] = true
		metadata["anthropic_base_url"] = options.AnthropicBaseURL
		return next, "Claude Cachy integration configured", nil
	case ClaudeUninstall:
		metadata, managed := next["cachy"].(map[string]any)
		if managed {
			if previousManagedURL, _ := metadata["anthropic_base_url"].(string); previousManagedURL != "" {
				if env, ok := next["env"].(map[string]any); ok {
					if currentURL, _ := env["ANTHROPIC_BASE_URL"].(string); currentURL == previousManagedURL {
						delete(env, "ANTHROPIC_BASE_URL")
					}
					if len(env) == 0 {
						delete(next, "env")
					}
				}
			}
		}
		delete(next, "cachy")
		return next, "Claude Cachy integration removed", nil
	default:
		return nil, "", fmt.Errorf("unsupported Claude integration action %q", options.Action)
	}
}

func parseJSONMap(text string) (map[string]any, error) {
	if text == "" {
		return map[string]any{}, nil
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return nil, fmt.Errorf("parse Claude settings: %w", err)
	}
	if value == nil {
		value = map[string]any{}
	}
	return value, nil
}

func childMap(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	value := map[string]any{}
	parent[key] = value
	return value
}

func cloneJSONMap(value map[string]any) map[string]any {
	clone := map[string]any{}
	for key, item := range value {
		if child, ok := item.(map[string]any); ok {
			clone[key] = cloneJSONMap(child)
			continue
		}
		clone[key] = item
	}
	return clone
}

func marshalSettings(value map[string]any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func normalizeJSON(text string) string {
	value, err := parseJSONMap(text)
	if err != nil {
		return text
	}
	normalized, err := marshalSettings(value)
	if err != nil {
		return text
	}
	return normalized
}

func validateBaseURL(raw, label string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid %s %q", label, raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid %s scheme %q", label, parsed.Scheme)
	}
	return nil
}
