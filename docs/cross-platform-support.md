# Cross-Platform Support

Cachy should treat Windows, macOS, and Linux as first-class platforms from the beginning.

## Platform Targets

Minimum supported targets:

```text
windows/amd64
windows/arm64
darwin/amd64
darwin/arm64
linux/amd64
linux/arm64
```

The Go CLI and proxy should build for all targets. The Electron app can support the same targets
where Electron packaging is practical.

## Current CLI Foundation Readiness

The CLI/platform foundation is ready for the next implementation phases when all of these are true:

- `cachy proxy` starts the transparent provider proxy from an explicit `--target` value or
  `CACHY_TARGET_BASE_URL`.
- `cachy doctor` reports version, config path, target URL, listen port, provider reachability, and
  credential presence without printing secret values.
- Platform path resolution returns config, state, and cache directories for Windows, macOS, and
  Linux without hard-coded Unix assumptions.
- Proxy logs use structured `log/slog` output with prompt, authorization, cookie, and API-key
  values redacted.
- CI runs formatting, vet, tests, race tests, and cross-compilation for every supported CLI target.

Run the current development CLI with:

```text
cachy proxy --listen 127.0.0.1:8787 --target http://127.0.0.1:11434
cachy doctor --target http://127.0.0.1:11434
```

The legacy `cachy-proxy` entrypoint remains available for compatibility while downstream docs and
recipes move to the unified `cachy` command.

## Core Rule

The Go binary must work without Electron.

Electron is a companion app, not a requirement. This keeps servers, dev containers, SSH sessions,
and headless Linux machines fully supported.

## Go Runtime Choices

Prefer dependencies that keep builds simple:

- Use pure-Go libraries where practical.
- Avoid CGO in the default build.
- Use `modernc.org/sqlite` before `mattn/go-sqlite3` if SQLite is needed in the default binary.
- Use `wazero` for WASM plugins because it is pure Go.
- Use `net/http`, `log/slog`, and other standard library packages where they are enough.

Optional CGO or native integrations can exist behind build tags, but the default install should not
require a compiler toolchain.

## OS-Specific Integration Strategy

### Windows

- No Unix hooks.
- Prefer config-file, MCP, environment, and wrapper-based integrations.
- Do not install Serena or other external semantic-code services as part of the default flow.
- Support PowerShell installer scripts.
- Package with `.exe`, ZIP, Winget, and Scoop.
- Store config under `%APPDATA%\Cachy` or `%LOCALAPPDATA%\Cachy`.
- Store secrets in Windows Credential Manager when available.
- Optional auto-start through Startup folder or Task Scheduler.

### macOS

- Support both Apple Silicon and Intel.
- Package with `.tar.gz`, Homebrew, and signed/notarized Electron builds later.
- Store config under `~/Library/Application Support/Cachy`.
- Store secrets in Keychain when available.
- Optional auto-start through LaunchAgent.

### Linux

- Support common distros through static-ish binaries and Docker.
- Package with `.tar.gz`, `.deb`, `.rpm`, and Homebrew-on-Linux later.
- Store config under `$XDG_CONFIG_HOME/cachy` or `~/.config/cachy`.
- Store state under `$XDG_STATE_HOME/cachy` or `~/.local/state/cachy`.
- Optional auto-start through systemd user service.

## File Layout

Use platform-aware helpers:

```text
config dir
  config.toml
  admin-token

state dir
  cachy.db
  logs/
  ccr/

cache dir
  plugins/
  downloads/
```

Never hard-code Unix paths like `/tmp` or `~/.config` without platform handling.

Default path resolution:

```text
Windows config : %APPDATA%\Cachy
Windows state  : %LOCALAPPDATA%\Cachy
Windows cache  : %LOCALAPPDATA%\Cachy\cache

macOS config   : ~/Library/Application Support/Cachy
macOS state    : ~/Library/Application Support/Cachy
macOS cache    : ~/Library/Caches/Cachy

Linux config   : $XDG_CONFIG_HOME/cachy or ~/.config/cachy
Linux state    : $XDG_STATE_HOME/cachy or ~/.local/state/cachy
Linux cache    : $XDG_CACHE_HOME/cachy or ~/.cache/cachy
```

## Packaging

### CLI

Release artifacts:

```text
cachy_windows_amd64.zip
cachy_windows_arm64.zip
cachy_darwin_amd64.tar.gz
cachy_darwin_arm64.tar.gz
cachy_linux_amd64.tar.gz
cachy_linux_arm64.tar.gz
```

The release workflow packages Windows builds as ZIP archives and macOS/Linux builds as tar.gz
archives. Each archive is uploaded with a matching `.sha256` sidecar.

### Docker

Publish multi-arch images:

```text
linux/amd64
linux/arm64
```

Docker is useful for servers and Windows users who prefer not to install local services. The release
workflow builds a non-published multi-arch OCI archive for `linux/amd64` and `linux/arm64`; pushing
to a public or private registry requires explicit owner approval.

### Electron

Electron should bundle the matching Cachy binary:

```text
win32-x64/cachy.exe
win32-arm64/cachy.exe
darwin-x64/cachy
darwin-arm64/cachy
linux-x64/cachy
linux-arm64/cachy
```

The app should discover a separately installed `cachy` binary first, then fall back to the bundled
one.

## CI Matrix

Every PR should run:

```text
go test ./...
go test -race ./...          # at least on linux/amd64
go vet ./...
gofmt check
cross-compile all targets
```

Recommended GitHub/Gitea workflow matrix:

```text
windows-latest
macos-latest
ubuntu-latest
```

The release build workflow compiles the unified `cachy` CLI for all six required targets on manual
dispatch and `v*` tags, then wraps those binaries into platform archives with checksums. Release
evidence generation writes SBOM JSON for source, CLI archives, and the Docker OCI artifact where
practical, plus release notes with verification steps and known limitations.

## Compatibility Tests

Required early tests:

- Windows path handling.
- Unix path handling.
- Config directory resolution.
- CLI proxy command defaults and target validation.
- CLI doctor diagnostics and secret-safe output.
- Privacy-safe proxy log redaction.
- Start proxy on random port.
- Health check.
- Archive extraction, PATH lookup, `cachy doctor`, proxy start, and `/healthz` install smoke test.
- OpenAI-compatible pass-through.
- Anthropic-compatible pass-through.
- SSE pass-through.
- Large body pass-through.
- Graceful shutdown.

Transparent proxy pass-through coverage and current limitations are tracked in
[Transparent Proxy MVP Readiness](transparent-proxy-mvp.md).

Later tests:

- Codex install on Windows/macOS/Linux.
- Claude install on Windows/macOS/Linux.
- Electron app launches bundled binary.
- WASM plugin host works on all platforms.

## Avoid

- Shell-script-only install flows.
- POSIX-only hooks as the primary integration path.
- Dependencies that need MSVC, Rust, Python, or GCC for normal users.
- Platform-specific path assumptions.
- In-process native extensions for the default runtime.
- Electron-only functionality that the CLI cannot perform.

## Platform Promise

A user should be able to download one archive, put `cachy` on PATH, and run:

```text
cachy doctor
cachy proxy --listen 127.0.0.1:8787
```

on Windows, macOS, or Linux without installing Python, Rust, Node, or a compiler.
