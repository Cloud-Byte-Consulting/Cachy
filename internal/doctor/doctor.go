package doctor

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cloud-byte-consulting/cachy/internal/platform"
)

type Status string

const (
	StatusOK   Status = "OK"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
)

type Config struct {
	Version            string
	TargetURL          string
	ListenAddress      string
	Env                map[string]string
	Paths              platform.Paths
	PathError          error
	CodexConfigPath    string
	ClaudeSettingsPath string
	ReadFile           func(string) ([]byte, error)
	CheckListen        func(string) error
	CheckReach         func(string) error
}

type Check struct {
	Name    string
	Status  Status
	Message string
}

type Report struct {
	Checks []Check
}

func RunChecks(config Config) Report {
	checks := []Check{
		checkVersion(config.Version),
		checkPaths(config.Paths, config.PathError),
		checkTargetURL(config.TargetURL),
		checkListen(config.ListenAddress, config.CheckListen),
		checkReachability(config.TargetURL, config.CheckReach),
		checkCredentials(config.Env),
		checkCodexIntegration(config.CodexConfigPath, config.ReadFile),
		checkClaudeIntegration(config.ClaudeSettingsPath, config.ReadFile),
		checkMCPRegistration(config.TargetURL, config.ListenAddress),
	}
	return Report{Checks: checks}
}

func (r Report) String() string {
	var out strings.Builder
	for _, check := range r.Checks {
		_, _ = fmt.Fprintf(&out, "%s %s: %s\n", check.Status, check.Name, check.Message)
	}
	return out.String()
}

func (r Report) HasFailures() bool {
	for _, check := range r.Checks {
		if check.Status == StatusFail {
			return true
		}
	}
	return false
}

func checkVersion(version string) Check {
	if version == "" {
		return Check{Name: "version", Status: StatusWarn, Message: "version is not set"}
	}
	return Check{Name: "version", Status: StatusOK, Message: version}
}

func checkPaths(paths platform.Paths, err error) Check {
	if err != nil {
		return Check{Name: "config-path", Status: StatusFail, Message: err.Error()}
	}
	if paths.ConfigDir == "" || paths.StateDir == "" || paths.CacheDir == "" {
		return Check{Name: "config-path", Status: StatusFail, Message: "config, state, and cache paths are required"}
	}
	return Check{Name: "config-path", Status: StatusOK, Message: paths.ConfigDir}
}

func checkTargetURL(raw string) Check {
	target, err := parseTargetURL(raw)
	if err != nil {
		return Check{Name: "target-url", Status: StatusFail, Message: err.Error()}
	}
	return Check{Name: "target-url", Status: StatusOK, Message: redactURL(target).String()}
}

func checkListen(address string, check func(string) error) Check {
	if address == "" {
		return Check{Name: "listen-port", Status: StatusFail, Message: "listen address is required"}
	}
	if check == nil {
		check = DefaultCheckListen
	}
	if err := check(address); err != nil {
		return Check{Name: "listen-port", Status: StatusFail, Message: err.Error()}
	}
	return Check{Name: "listen-port", Status: StatusOK, Message: address}
}

func checkReachability(raw string, check func(string) error) Check {
	if _, err := parseTargetURL(raw); err != nil {
		return Check{Name: "provider-reachability", Status: StatusFail, Message: "target URL is invalid"}
	}
	if check == nil {
		check = DefaultCheckReachability
	}
	if err := check(raw); err != nil {
		return Check{Name: "provider-reachability", Status: StatusFail, Message: err.Error()}
	}
	return Check{Name: "provider-reachability", Status: StatusOK, Message: "target responded"}
}

func checkCredentials(env map[string]string) Check {
	for _, key := range []string{"CACHY_PROVIDER_API_KEY", "OPENAI_API_KEY", "ANTHROPIC_API_KEY"} {
		if envValue(env, key) != "" {
			return Check{Name: "credentials", Status: StatusOK, Message: key + " is set"}
		}
	}
	return Check{Name: "credentials", Status: StatusWarn, Message: "no provider credential found in environment"}
}

func checkCodexIntegration(path string, readFile func(string) ([]byte, error)) Check {
	name := "codex-integration"
	if path == "" {
		return Check{Name: name, Status: StatusWarn, Message: "Codex config path unavailable; set CODEX_HOME or HOME and run cachy integrations codex install --dry-run"}
	}
	data, err := readIntegrationFile(path, readFile)
	if errors.Is(err, os.ErrNotExist) {
		return Check{Name: name, Status: StatusWarn, Message: "Codex integration is not configured; run cachy integrations codex install --dry-run --config " + path}
	}
	if err != nil {
		return Check{Name: name, Status: StatusWarn, Message: "Codex config could not be read; run cachy integrations codex repair --dry-run --config " + path}
	}

	text := string(data)
	hasBegin := strings.Contains(text, "# BEGIN CACHY CODEX INTEGRATION")
	hasEnd := strings.Contains(text, "# END CACHY CODEX INTEGRATION")
	if hasBegin != hasEnd {
		return Check{Name: name, Status: StatusWarn, Message: "Codex managed block is incomplete; run cachy integrations codex repair --dry-run --config " + path}
	}
	if !hasBegin {
		return Check{Name: name, Status: StatusWarn, Message: "Codex integration is not configured; run cachy integrations codex install --dry-run --config " + path}
	}
	for _, required := range []string{`[model_providers.cachy]`, `base_url =`, `env_key = "OPENAI_API_KEY"`, `wire_api = "responses"`} {
		if !strings.Contains(text, required) {
			return Check{Name: name, Status: StatusWarn, Message: "Codex managed block is incomplete; run cachy integrations codex repair --dry-run --config " + path}
		}
	}
	return Check{Name: name, Status: StatusOK, Message: "configured in " + path + "; verify changes with cachy integrations codex repair --dry-run --config " + path}
}

func checkClaudeIntegration(path string, readFile func(string) ([]byte, error)) Check {
	name := "claude-integration"
	if path == "" {
		return Check{Name: name, Status: StatusWarn, Message: "Claude settings path unavailable; set CLAUDE_HOME or HOME and run cachy integrations claude install --dry-run"}
	}
	data, err := readIntegrationFile(path, readFile)
	if errors.Is(err, os.ErrNotExist) {
		return Check{Name: name, Status: StatusWarn, Message: "Claude integration is not configured; run cachy integrations claude install --dry-run --settings " + path}
	}
	if err != nil {
		return Check{Name: name, Status: StatusWarn, Message: "Claude settings could not be read; run cachy integrations claude repair --dry-run --settings " + path}
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return Check{Name: name, Status: StatusWarn, Message: "Claude settings are not valid JSON; run cachy integrations claude repair --dry-run --settings " + path}
	}
	env, _ := settings["env"].(map[string]any)
	metadata, _ := settings["cachy"].(map[string]any)
	managed, _ := metadata["managed"].(bool)
	currentURL, _ := env["ANTHROPIC_BASE_URL"].(string)
	managedURL, _ := metadata["anthropic_base_url"].(string)
	if !managed || currentURL == "" || managedURL == "" || currentURL != managedURL {
		return Check{Name: name, Status: StatusWarn, Message: "Claude Cachy settings are incomplete; run cachy integrations claude repair --dry-run --settings " + path}
	}
	return Check{Name: name, Status: StatusOK, Message: "configured in " + path + "; verify changes with cachy integrations claude repair --dry-run --settings " + path}
}

func checkMCPRegistration(rawTarget, listen string) Check {
	name := "mcp-registration"
	target, err := parseTargetURL(rawTarget)
	if err != nil {
		return Check{Name: name, Status: StatusWarn, Message: "MCP registration cannot be generated until target URL is valid"}
	}
	if listen == "" {
		return Check{Name: name, Status: StatusWarn, Message: "MCP registration cannot be generated until listen address is set"}
	}
	return Check{Name: name, Status: StatusOK, Message: "generate dry-run registration with cachy integrations mcp --target " + redactURL(target).String() + " --listen " + listen}
}

func readIntegrationFile(path string, readFile func(string) ([]byte, error)) ([]byte, error) {
	if readFile == nil {
		readFile = os.ReadFile
	}
	return readFile(path)
}

func parseTargetURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, errors.New("target URL is required")
	}
	target, err := url.Parse(raw)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return nil, errors.New("target URL must include scheme and host")
	}
	return target, nil
}

func DefaultCheckListen(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	return listener.Close()
}

func DefaultCheckReachability(raw string) error {
	client := http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodHead, raw, nil)
	if err != nil {
		return sanitizeURLError(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return sanitizeURLError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("target returned status %d", resp.StatusCode)
	}
	return nil
}

func envValue(env map[string]string, key string) string {
	if env != nil {
		return env[key]
	}
	return ""
}

func sanitizeURLError(err error) error {
	if err == nil {
		return nil
	}

	var urlErr *url.Error
	if !errors.As(err, &urlErr) || urlErr.URL == "" {
		return err
	}

	cleanURL := urlErr.URL
	if parsed, parseErr := url.Parse(urlErr.URL); parseErr == nil {
		cleanURL = redactURL(parsed).String()
	}
	return &url.Error{Op: urlErr.Op, URL: cleanURL, Err: urlErr.Err}
}

func redactURL(value *url.URL) *url.URL {
	if value == nil {
		return nil
	}
	clean := *value
	if clean.User != nil {
		clean.User = url.UserPassword("redacted", "redacted")
	}
	return &clean
}
