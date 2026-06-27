package install

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	ManagedCodexBegin = "# BEGIN CACHY CODEX INTEGRATION"
	ManagedCodexEnd   = "# END CACHY CODEX INTEGRATION"

	defaultCodexProxyBaseURL = "http://127.0.0.1:8787/v1"
)

type CodexAction string

const (
	CodexInstall   CodexAction = "install"
	CodexRepair    CodexAction = "repair"
	CodexUninstall CodexAction = "uninstall"
)

type CodexOptions struct {
	Action       CodexAction
	ConfigPath   string
	ProxyBaseURL string
	DryRun       bool
}

type CodexResult struct {
	Action  CodexAction
	DryRun  bool
	Changed bool
	Path    string
	Preview string
	Message string
}

func DefaultCodexConfigPath(homeDir, codexHome string) (string, error) {
	if codexHome != "" {
		return filepath.Join(codexHome, "config.toml"), nil
	}
	if homeDir == "" {
		return "", errors.New("home directory is required to resolve Codex config")
	}
	return filepath.Join(homeDir, ".codex", "config.toml"), nil
}

func ApplyCodexIntegration(options CodexOptions) (CodexResult, error) {
	if options.ConfigPath == "" {
		return CodexResult{}, errors.New("Codex config path is required")
	}
	if options.Action == "" {
		options.Action = CodexInstall
	}
	if options.ProxyBaseURL == "" {
		options.ProxyBaseURL = defaultCodexProxyBaseURL
	}
	if options.Action != CodexUninstall {
		if err := validateProxyBaseURL(options.ProxyBaseURL); err != nil {
			return CodexResult{}, err
		}
	}

	current, err := readOptionalFile(options.ConfigPath)
	if err != nil {
		return CodexResult{}, err
	}
	next, message, err := planCodexConfig(current, options)
	if err != nil {
		return CodexResult{}, err
	}

	result := CodexResult{
		Action:  options.Action,
		DryRun:  options.DryRun,
		Changed: next != current,
		Path:    options.ConfigPath,
		Preview: next,
		Message: message,
	}
	if options.DryRun || !result.Changed {
		return result, nil
	}
	if err := os.MkdirAll(filepath.Dir(options.ConfigPath), 0o700); err != nil {
		return CodexResult{}, err
	}
	if err := os.WriteFile(options.ConfigPath, []byte(next), 0o600); err != nil {
		return CodexResult{}, err
	}
	return result, nil
}

func planCodexConfig(current string, options CodexOptions) (string, string, error) {
	cleaned := removeManagedBlock(current)
	switch options.Action {
	case CodexInstall, CodexRepair:
		block := renderCodexManagedBlock(options.ProxyBaseURL, hasTopLevelModelProvider(cleaned))
		next := appendBlock(cleaned, block)
		message := "Codex Cachy integration configured"
		if hasTopLevelModelProvider(cleaned) {
			message = "Codex provider block configured; existing model_provider preserved"
		}
		return next, message, nil
	case CodexUninstall:
		return cleaned, "Codex Cachy integration removed", nil
	default:
		return "", "", fmt.Errorf("unsupported Codex integration action %q", options.Action)
	}
}

func renderCodexManagedBlock(proxyBaseURL string, preserveProvider bool) string {
	lines := []string{
		ManagedCodexBegin,
		"# Managed by Cachy. Remove with: cachy integrations codex uninstall",
	}
	if !preserveProvider {
		lines = append(lines, `model_provider = "cachy"`)
	}
	lines = append(lines,
		`[model_providers.cachy]`,
		`name = "Cachy"`,
		fmt.Sprintf(`base_url = %q`, proxyBaseURL),
		`env_key = "OPENAI_API_KEY"`,
		`wire_api = "responses"`,
		ManagedCodexEnd,
	)
	return strings.Join(lines, "\n") + "\n"
}

func removeManagedBlock(text string) string {
	start := strings.Index(text, ManagedCodexBegin)
	if start < 0 {
		return text
	}
	end := strings.Index(text[start:], ManagedCodexEnd)
	if end < 0 {
		return text[:start]
	}
	end = start + end + len(ManagedCodexEnd)
	if end < len(text) && text[end] == '\r' {
		end++
	}
	if end < len(text) && text[end] == '\n' {
		end++
	}
	return strings.TrimRight(text[:start], "\r\n") + trailingNewline(text[:start]) + strings.TrimLeft(text[end:], "\r\n")
}

func appendBlock(current, block string) string {
	current = strings.TrimRight(current, "\r\n")
	if current == "" {
		return block
	}
	return current + "\n\n" + block
}

func trailingNewline(prefix string) string {
	if strings.TrimSpace(prefix) == "" {
		return ""
	}
	return "\n"
}

func hasTopLevelModelProvider(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "[") {
			continue
		}
		if strings.HasPrefix(trimmed, "model_provider") && strings.Contains(trimmed, "=") {
			return true
		}
	}
	return false
}

func readOptionalFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func validateProxyBaseURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid Codex proxy base URL %q", raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid Codex proxy base URL scheme %q", parsed.Scheme)
	}
	return nil
}
