package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/doctor"
	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/install"
	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/observability"
	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/platform"
	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/plugin"
	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/proxy"
)

const DefaultListenAddress = "127.0.0.1:8787"

type EnvFunc func(string) string

type ProxyConfig struct {
	Listen        string
	TargetBaseURL string
}

type ProxyServeFunc func(ProxyConfig) error

type Deps struct {
	Getenv     EnvFunc
	Stdout     io.Writer
	Stderr     io.Writer
	ServeProxy ProxyServeFunc
	RunDoctor  func(doctor.Config) doctor.Report
	RunPlugin  plugin.HostRunner
}

func Run(args []string, getenv EnvFunc, stdout, stderr io.Writer, serveProxy ProxyServeFunc) int {
	return RunWithDeps(args, Deps{
		Getenv:     getenv,
		Stdout:     stdout,
		Stderr:     stderr,
		ServeProxy: serveProxy,
	})
}

func RunWithDeps(args []string, deps Deps) int {
	getenv := deps.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	stdout := deps.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := deps.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	serveProxy := deps.ServeProxy
	if serveProxy == nil {
		serveProxy = ServeProxy
	}
	runDoctor := deps.RunDoctor

	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "proxy":
		return runProxy(args[1:], getenv, stderr, serveProxy)
	case "doctor":
		return runDoctorCommand(args[1:], getenv, stdout, stderr, runDoctor)
	case "integrations":
		return runIntegrationsCommand(args[1:], getenv, stdout, stderr)
	case "plugin":
		return runPluginCommand(args[1:], getenv, stdout, stderr, deps.RunPlugin)
	case "-h", "--help", "help":
		printUsage(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runPluginCommand(args []string, getenv EnvFunc, stdout, stderr io.Writer, runner plugin.HostRunner) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: cachy plugin <list|inspect|enable|disable|test> [options]")
		return 2
	}

	switch args[0] {
	case "list":
		flags := flag.NewFlagSet("cachy plugin list", flag.ContinueOnError)
		flags.SetOutput(stderr)
		dir := flags.String("dir", defaultPluginDir(getenv), "plugin directory")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		plugins, err := plugin.ListPlugins(*dir)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "plugin list failed: %v\n", err)
			return 1
		}
		for _, info := range plugins {
			_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\n", info.Manifest.Name, enabledLabel(info.Enabled), strings.Join(info.Manifest.Capabilities.Compress, ","))
		}
		return 0
	case "inspect":
		name, remaining, ok := pluginNameArg(args[1:], stderr, "inspect")
		if !ok {
			return 2
		}
		flags := flag.NewFlagSet("cachy plugin inspect", flag.ContinueOnError)
		flags.SetOutput(stderr)
		dir := flags.String("dir", defaultPluginDir(getenv), "plugin directory")
		if err := flags.Parse(remaining); err != nil {
			return 2
		}
		info, err := plugin.InspectPlugin(*dir, name)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "plugin inspect failed: %v\n", err)
			return 1
		}
		printPluginInfo(stdout, info)
		return 0
	case "enable", "disable":
		name, remaining, ok := pluginNameArg(args[1:], stderr, args[0])
		if !ok {
			return 2
		}
		flags := flag.NewFlagSet("cachy plugin "+args[0], flag.ContinueOnError)
		flags.SetOutput(stderr)
		dir := flags.String("dir", defaultPluginDir(getenv), "plugin directory")
		if err := flags.Parse(remaining); err != nil {
			return 2
		}
		if err := plugin.SetEnabled(*dir, name, args[0] == "enable"); err != nil {
			_, _ = fmt.Fprintf(stderr, "plugin %s failed: %v\n", args[0], err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "%s %s\n", name, enabledLabel(args[0] == "enable"))
		return 0
	case "test":
		name, remaining, ok := pluginNameArg(args[1:], stderr, "test")
		if !ok {
			return 2
		}
		flags := flag.NewFlagSet("cachy plugin test", flag.ContinueOnError)
		flags.SetOutput(stderr)
		dir := flags.String("dir", defaultPluginDir(getenv), "plugin directory")
		fixture := flags.String("fixture", "", "path to a text fixture to send as the selected block")
		if err := flags.Parse(remaining); err != nil {
			return 2
		}
		if *fixture == "" {
			_, _ = fmt.Fprintln(stderr, "missing --fixture")
			return 2
		}
		result, err := plugin.TestPlugin(context.Background(), plugin.TestOptions{
			RootDir:     *dir,
			Name:        name,
			FixturePath: *fixture,
			Runner:      runner,
		})
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "plugin test failed: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "plugin: %s\naction: %s\ntext_bytes: %d\nsummary: %s\n", result.Plugin.Manifest.Name, result.Output.Action, len(result.Output.Text), result.Output.Summary)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown plugin command %q\n", args[0])
		return 2
	}
}

func pluginNameArg(args []string, stderr io.Writer, command string) (string, []string, bool) {
	if len(args) == 0 {
		_, _ = fmt.Fprintf(stderr, "usage: cachy plugin %s <name> [options]\n", command)
		return "", nil, false
	}
	return args[0], args[1:], true
}

func printPluginInfo(stdout io.Writer, info plugin.Info) {
	_, _ = fmt.Fprintf(stdout, "name: %s\n", info.Manifest.Name)
	_, _ = fmt.Fprintf(stdout, "version: %s\n", info.Manifest.Version)
	_, _ = fmt.Fprintf(stdout, "api_version: %s\n", info.Manifest.APIVersion)
	_, _ = fmt.Fprintf(stdout, "status: %s\n", enabledLabel(info.Enabled))
	_, _ = fmt.Fprintf(stdout, "compress: %s\n", strings.Join(info.Manifest.Capabilities.Compress, ","))
	_, _ = fmt.Fprintf(stdout, "timeout_ms: %d\n", info.Manifest.Limits.TimeoutMS)
	_, _ = fmt.Fprintf(stdout, "memory_mb: %d\n", info.Manifest.Limits.MemoryMB)
}

func enabledLabel(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func defaultPluginDir(getenv EnvFunc) string {
	if getenv == nil {
		getenv = os.Getenv
	}
	if dir := getenv("CACHY_PLUGIN_DIR"); dir != "" {
		return dir
	}
	paths, err := platform.ResolvePaths(platform.Options{Env: collectDoctorEnv(getenv)})
	if err != nil {
		return "plugins"
	}
	return filepath.Join(paths.StateDir, "plugins")
}

func runIntegrationsCommand(args []string, getenv EnvFunc, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		_, _ = fmt.Fprintln(stderr, "usage: cachy integrations <codex|claude|recipe|mcp> ...")
		return 2
	}
	switch args[0] {
	case "codex":
		return runCodexIntegrationCommand(args[1:], getenv, stdout, stderr)
	case "claude":
		return runClaudeIntegrationCommand(args[1:], getenv, stdout, stderr)
	case "recipe":
		return runRecipeCommand(args[1:], stdout, stderr)
	case "mcp":
		return runMCPCommand(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown integration %q\n", args[0])
		return 2
	}
}

func runCodexIntegrationCommand(args []string, getenv EnvFunc, stdout, stderr io.Writer) int {
	if getenv == nil {
		getenv = os.Getenv
	}
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: cachy integrations codex <install|repair|uninstall> [options]")
		return 2
	}

	action := install.CodexAction(args[0])
	flags := flag.NewFlagSet("cachy integrations codex "+string(action), flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "path to Codex config.toml")
	proxyBaseURL := flags.String("proxy-base-url", "http://127.0.0.1:8787/v1", "Cachy OpenAI-compatible base URL")
	dryRun := flags.Bool("dry-run", false, "print intended changes without writing")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}

	resolvedConfigPath := *configPath
	if resolvedConfigPath == "" {
		var err error
		resolvedConfigPath, err = install.DefaultCodexConfigPath(getenv("HOME"), getenv("CODEX_HOME"))
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "resolve Codex config: %v\n", err)
			return 2
		}
	}
	result, err := install.ApplyCodexIntegration(install.CodexOptions{
		Action:       action,
		ConfigPath:   resolvedConfigPath,
		ProxyBaseURL: *proxyBaseURL,
		DryRun:       *dryRun,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Codex integration failed: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s\npath: %s\nchanged: %v\n", result.Message, result.Path, result.Changed)
	if result.DryRun {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprint(stdout, result.Preview)
	}
	return 0
}

func runRecipeCommand(args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: cachy integrations recipe <name|custom> [options]")
		return 2
	}
	flags := flag.NewFlagSet("cachy integrations recipe", flag.ContinueOnError)
	flags.SetOutput(stderr)
	cachyBaseURL := flags.String("cachy-base-url", "http://127.0.0.1:8787", "Cachy proxy base URL")
	target := flags.String("target", "", "custom OpenAI-compatible target URL")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}

	var (
		recipe install.EndpointRecipe
		err    error
	)
	if args[0] == "custom" {
		recipe, err = install.CustomEndpointRecipe(*target)
	} else {
		recipe, err = install.EndpointRecipeByName(args[0])
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "recipe failed: %v\n", err)
		return 1
	}
	rendered, err := install.RenderEndpointRecipe(recipe, install.RecipeOptions{CachyBaseURL: *cachyBaseURL})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "recipe failed: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprint(stdout, rendered)
	return 0
}

func runMCPCommand(args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	flags := flag.NewFlagSet("cachy integrations mcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	command := flags.String("command", "cachy", "Cachy command path")
	listen := flags.String("listen", DefaultListenAddress, "Cachy proxy listen address")
	target := flags.String("target", "", "OpenAI-compatible target URL")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	rendered, err := install.RenderMCPRegistration(install.MCPOptions{
		Command:       *command,
		ListenAddress: *listen,
		TargetBaseURL: *target,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "MCP registration failed: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprint(stdout, rendered)
	return 0
}

func runClaudeIntegrationCommand(args []string, getenv EnvFunc, stdout, stderr io.Writer) int {
	if getenv == nil {
		getenv = os.Getenv
	}
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: cachy integrations claude <install|repair|uninstall> [options]")
		return 2
	}

	action := install.ClaudeAction(args[0])
	flags := flag.NewFlagSet("cachy integrations claude "+string(action), flag.ContinueOnError)
	flags.SetOutput(stderr)
	settingsPath := flags.String("settings", "", "path to Claude settings.json")
	anthropicBaseURL := flags.String("anthropic-base-url", "http://127.0.0.1:8787", "Cachy Anthropic-compatible base URL")
	dryRun := flags.Bool("dry-run", false, "print intended changes without writing")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}

	resolvedSettingsPath := *settingsPath
	if resolvedSettingsPath == "" {
		var err error
		resolvedSettingsPath, err = install.DefaultClaudeSettingsPath(getenv("HOME"), getenv("CLAUDE_HOME"))
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "resolve Claude settings: %v\n", err)
			return 2
		}
	}
	result, err := install.ApplyClaudeIntegration(install.ClaudeOptions{
		Action:           action,
		SettingsPath:     resolvedSettingsPath,
		AnthropicBaseURL: *anthropicBaseURL,
		DryRun:           *dryRun,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Claude integration failed: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s\npath: %s\nchanged: %v\n", result.Message, result.Path, result.Changed)
	if result.DryRun {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprint(stdout, result.Preview)
	}
	return 0
}

func RunProxy(args []string, getenv EnvFunc, stderr io.Writer, serveProxy ProxyServeFunc) int {
	return runProxy(args, getenv, stderr, serveProxy)
}

func runProxy(args []string, getenv EnvFunc, stderr io.Writer, serveProxy ProxyServeFunc) int {
	if getenv == nil {
		getenv = os.Getenv
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if serveProxy == nil {
		serveProxy = ServeProxy
	}

	flags := flag.NewFlagSet("cachy proxy", flag.ContinueOnError)
	flags.SetOutput(stderr)
	listen := flags.String("listen", DefaultListenAddress, "address for the proxy to listen on")
	target := flags.String("target", getenv("CACHY_TARGET_BASE_URL"), "upstream provider base URL")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *target == "" {
		_, _ = fmt.Fprintln(stderr, "missing upstream target: set --target or CACHY_TARGET_BASE_URL")
		return 2
	}

	if err := serveProxy(ProxyConfig{Listen: *listen, TargetBaseURL: *target}); err != nil {
		_, _ = fmt.Fprintf(stderr, "proxy stopped: %v\n", err)
		return 1
	}
	return 0
}

func runDoctorCommand(args []string, getenv EnvFunc, stdout, stderr io.Writer, runDoctor func(doctor.Config) doctor.Report) int {
	if getenv == nil {
		getenv = os.Getenv
	}
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if runDoctor == nil {
		runDoctor = doctor.RunChecks
	}

	flags := flag.NewFlagSet("cachy doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	listen := flags.String("listen", DefaultListenAddress, "address for the proxy to listen on")
	target := flags.String("target", getenv("CACHY_TARGET_BASE_URL"), "upstream provider base URL")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	env := collectDoctorEnv(getenv)
	paths, pathErr := platform.ResolvePaths(platform.Options{Env: env})
	homeDir := homeFromEnv(env)
	codexConfigPath, _ := install.DefaultCodexConfigPath(homeDir, env["CODEX_HOME"])
	claudeSettingsPath, _ := install.DefaultClaudeSettingsPath(homeDir, env["CLAUDE_HOME"])
	report := runDoctor(doctor.Config{
		Version:            "dev",
		TargetURL:          *target,
		ListenAddress:      *listen,
		Env:                env,
		Paths:              paths,
		PathError:          pathErr,
		CodexConfigPath:    codexConfigPath,
		ClaudeSettingsPath: claudeSettingsPath,
	})
	_, _ = fmt.Fprint(stdout, report.String())
	if report.HasFailures() {
		return 1
	}
	return 0
}

func homeFromEnv(env map[string]string) string {
	if env["HOME"] != "" {
		return env["HOME"]
	}
	return env["USERPROFILE"]
}

func ServeProxy(config ProxyConfig) error {
	handler, err := proxy.New(proxy.Config{
		TargetBaseURL: config.TargetBaseURL,
		Logger:        slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{ReplaceAttr: observability.RedactAttr})),
	})
	if err != nil {
		return fmt.Errorf("failed to configure proxy: %w", err)
	}

	server := &http.Server{
		Addr:              config.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage: cachy <command> [options]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "commands:")
	_, _ = fmt.Fprintln(w, "  proxy    start the transparent provider proxy")
	_, _ = fmt.Fprintln(w, "  doctor   run local setup diagnostics")
	_, _ = fmt.Fprintln(w, "  integrations <codex|claude|recipe|mcp> ...  manage agent setup")
	_, _ = fmt.Fprintln(w, "  plugin <list|inspect|enable|disable|test> ...  manage WASM plugins")
}

func collectDoctorEnv(getenv EnvFunc) map[string]string {
	keys := []string{
		"CACHY_TARGET_BASE_URL",
		"CACHY_PROVIDER_API_KEY",
		"OPENAI_API_KEY",
		"ANTHROPIC_API_KEY",
		"HOME",
		"USERPROFILE",
		"APPDATA",
		"LOCALAPPDATA",
		"XDG_CONFIG_HOME",
		"XDG_STATE_HOME",
		"XDG_CACHE_HOME",
		"CODEX_HOME",
		"CLAUDE_HOME",
	}
	env := make(map[string]string, len(keys))
	for _, key := range keys {
		env[key] = getenv(key)
	}
	return env
}
