package doctor

import (
	"errors"
	"os"
	"strings"
	"testing"

	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/platform"
)

func TestRunChecksReportsHealthySetupWithoutSecrets(t *testing.T) {
	t.Parallel()

	report := RunChecks(Config{
		Version:       "dev",
		TargetURL:     "http://127.0.0.1:11434",
		ListenAddress: "127.0.0.1:8787",
		Env: map[string]string{
			"OPENAI_API_KEY": "secret-key",
		},
		Paths: platform.Paths{
			ConfigDir: "/cfg/cachy",
			StateDir:  "/state/cachy",
			CacheDir:  "/cache/cachy",
		},
		CheckListen: func(string) error { return nil },
		CheckReach:  func(string) error { return nil },
	})

	assertCheck(t, report, "version", StatusOK)
	assertCheck(t, report, "config-path", StatusOK)
	assertCheck(t, report, "target-url", StatusOK)
	assertCheck(t, report, "listen-port", StatusOK)
	assertCheck(t, report, "provider-reachability", StatusOK)
	assertCheck(t, report, "credentials", StatusOK)

	output := report.String()
	if strings.Contains(output, "secret-key") {
		t.Fatalf("doctor output leaked credential: %s", output)
	}
}

func TestRunChecksReportsActionableFailures(t *testing.T) {
	t.Parallel()

	report := RunChecks(Config{
		Version:       "",
		TargetURL:     "://bad-url",
		ListenAddress: "127.0.0.1:8787",
		PathError:     errors.New("home directory unavailable"),
		CheckListen:   func(string) error { return errors.New("port busy") },
		CheckReach:    func(string) error { return errors.New("unreachable") },
	})

	assertCheck(t, report, "version", StatusWarn)
	assertCheck(t, report, "config-path", StatusFail)
	assertCheck(t, report, "target-url", StatusFail)
	assertCheck(t, report, "listen-port", StatusFail)
	assertCheck(t, report, "provider-reachability", StatusFail)
	assertCheck(t, report, "credentials", StatusWarn)
}

func TestReportStringIncludesStatusesAndMessages(t *testing.T) {
	t.Parallel()

	report := Report{Checks: []Check{
		{Name: "target-url", Status: StatusOK, Message: "http://127.0.0.1:11434"},
		{Name: "credentials", Status: StatusWarn, Message: "no provider credential found"},
	}}

	output := report.String()
	for _, want := range []string{"OK target-url", "WARN credentials", "no provider credential found"} {
		if !strings.Contains(output, want) {
			t.Fatalf("report output %q does not contain %q", output, want)
		}
	}
}

func TestDefaultCheckReachabilityRedactsURLCredentials(t *testing.T) {
	t.Parallel()

	err := DefaultCheckReachability("http://user:secret@127.0.0.1:1")
	if err == nil {
		t.Fatal("reachability check succeeded, want failure")
	}

	message := err.Error()
	if strings.Contains(message, "secret") || strings.Contains(message, "user:secret") {
		t.Fatalf("reachability error leaked URL credentials: %s", message)
	}
	if !strings.Contains(message, "redacted") {
		t.Fatalf("reachability error = %q, want redacted URL", message)
	}
}

func TestRunChecksReportsConfiguredIntegrations(t *testing.T) {
	t.Parallel()

	report := RunChecks(Config{
		Version:            "dev",
		TargetURL:          "http://user:secret@127.0.0.1:11434",
		ListenAddress:      "127.0.0.1:8787",
		CodexConfigPath:    "/home/alex/.codex/config.toml",
		ClaudeSettingsPath: "/home/alex/.claude/settings.json",
		Env: map[string]string{
			"OPENAI_API_KEY": "secret-key",
		},
		Paths: platform.Paths{
			ConfigDir: "/cfg/cachy",
			StateDir:  "/state/cachy",
			CacheDir:  "/cache/cachy",
		},
		ReadFile: func(path string) ([]byte, error) {
			switch path {
			case "/home/alex/.codex/config.toml":
				return []byte(`# BEGIN CACHY CODEX INTEGRATION
model_provider = "cachy"
[model_providers.cachy]
base_url = "http://127.0.0.1:8787/v1"
env_key = "OPENAI_API_KEY"
wire_api = "responses"
# END CACHY CODEX INTEGRATION
`), nil
			case "/home/alex/.claude/settings.json":
				return []byte(`{"env":{"ANTHROPIC_BASE_URL":"http://127.0.0.1:8787"},"cachy":{"managed":true,"anthropic_base_url":"http://127.0.0.1:8787"}}`), nil
			default:
				return nil, os.ErrNotExist
			}
		},
		CheckListen: func(string) error { return nil },
		CheckReach:  func(string) error { return nil },
	})

	assertCheck(t, report, "codex-integration", StatusOK)
	assertCheck(t, report, "claude-integration", StatusOK)
	assertCheck(t, report, "mcp-registration", StatusOK)

	output := report.String()
	for _, leaked := range []string{"secret-key", "user:secret", "secret@"} {
		if strings.Contains(output, leaked) {
			t.Fatalf("doctor output leaked secret %q:\n%s", leaked, output)
		}
	}
	if !strings.Contains(output, "cachy integrations codex repair --dry-run") {
		t.Fatalf("doctor output missing Codex dry-run repair guidance:\n%s", output)
	}
	if !strings.Contains(output, "cachy integrations claude repair --dry-run") {
		t.Fatalf("doctor output missing Claude dry-run repair guidance:\n%s", output)
	}
	if !strings.Contains(output, "cachy integrations mcp --target http://redacted:redacted@127.0.0.1:11434 --listen 127.0.0.1:8787") {
		t.Fatalf("doctor output missing redacted MCP guidance:\n%s", output)
	}
}

func TestRunChecksReportsMissingAndPartialIntegrations(t *testing.T) {
	t.Parallel()

	report := RunChecks(Config{
		Version:            "dev",
		TargetURL:          "http://127.0.0.1:11434",
		ListenAddress:      "127.0.0.1:8787",
		CodexConfigPath:    "/home/alex/.codex/config.toml",
		ClaudeSettingsPath: "/home/alex/.claude/settings.json",
		Paths: platform.Paths{
			ConfigDir: "/cfg/cachy",
			StateDir:  "/state/cachy",
			CacheDir:  "/cache/cachy",
		},
		ReadFile: func(path string) ([]byte, error) {
			switch path {
			case "/home/alex/.codex/config.toml":
				return []byte("# BEGIN CACHY CODEX INTEGRATION\n"), nil
			case "/home/alex/.claude/settings.json":
				return []byte(`{"cachy":{"managed":true}}`), nil
			default:
				return nil, os.ErrNotExist
			}
		},
		CheckListen: func(string) error { return nil },
		CheckReach:  func(string) error { return nil },
	})

	assertCheck(t, report, "codex-integration", StatusWarn)
	assertCheck(t, report, "claude-integration", StatusWarn)
	output := report.String()
	for _, want := range []string{
		"Codex managed block is incomplete",
		"Claude Cachy settings are incomplete",
		"cachy integrations codex repair --dry-run",
		"cachy integrations claude repair --dry-run",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, output)
		}
	}
}

func assertCheck(t *testing.T, report Report, name string, status Status) {
	t.Helper()

	for _, check := range report.Checks {
		if check.Name == name {
			if check.Status != status {
				t.Fatalf("%s status = %s, want %s", name, check.Status, status)
			}
			return
		}
	}
	t.Fatalf("check %q not found in %#v", name, report.Checks)
}
