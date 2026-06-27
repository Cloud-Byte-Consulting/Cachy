# Electron Companion App

Cachy should ship as a native Go CLI first, with an optional Electron companion app layered on top.
The desktop app should not replace the Go runtime. It should supervise it, configure it, and make the
local proxy understandable.

## Product Role

The app is a local dashboard and setup assistant:

- Start, stop, and restart the local Cachy proxy.
- Install or repair Codex and Claude integrations.
- Configure provider targets and API keys.
- Show live compression savings, cache hit signals, latency, and errors.
- Explain what was compressed and what was preserved.
- Manage local CCR retrieval storage.
- Run diagnostics for ports, env vars, auth, and provider connectivity.

The app should make Cachy feel trustworthy. Users should be able to see whether it is active, where
traffic is going, and what value it is producing.

## Architecture

```text
Electron app
  main process
    ├── owns tray/menu/window lifecycle
    ├── launches or discovers cachy binary
    ├── talks to local admin API
    └── handles auto-update

  renderer
    ├── dashboard
    ├── provider settings
    ├── integrations setup
    ├── sessions/logs
    └── diagnostics

Go binary
  ├── public proxy API       : 127.0.0.1:8787
  ├── private admin API      : 127.0.0.1:<random-port>
  ├── local SQLite state
  ├── metrics endpoint
  └── install/wrap commands
```

Electron should never need direct database writes. It should talk to Cachy through a private local
admin API or by invoking `cachy` commands.

## Go Runtime Responsibilities

- Proxy all model traffic.
- Own compression decisions.
- Own provider request forwarding.
- Own local persistence.
- Own security-sensitive config validation.
- Expose a small local admin API.
- Emit structured logs and metrics.

## Electron Responsibilities

- User onboarding.
- Visual status and diagnostics.
- Safe config editing.
- Provider setup forms.
- Agent integration installers.
- Auto-start preference.
- Update notifications.
- Local activity history.

The app should install Cachy integrations only. It should not install Serena or any unrelated
semantic-code companion by default.

## Admin API Sketch

The Go binary can expose a localhost-only admin API protected by a random token written to the user
config directory at startup.

Admin handlers must be constructed behind the shared `internal/admin` token guard. The guard rejects
non-loopback listen addresses, requires a non-empty generated or configured token, and accepts either
`Authorization: Bearer <token>` or `X-Cachy-Admin-Token: <token>`. Rejection responses must stay
generic and must not echo token values, header names, provider keys, or prompt content.

```text
GET  /admin/v1/status
GET  /admin/v1/config
PUT  /admin/v1/config
GET  /admin/v1/metrics
GET  /admin/v1/diagnostics
GET  /admin/v1/sessions
GET  /admin/v1/sessions/{id}
POST /admin/v1/proxy/start
POST /admin/v1/proxy/stop
POST /admin/v1/integrations/claude/install
POST /admin/v1/integrations/codex/install
POST /admin/v1/doctor/run
```

The initial status/config implementation returns runtime status, version, proxy listen address,
target configuration presence, and platform paths. Config reads expose the current target base URL
and redact provider credential presence as `<redacted>`. Config writes are intentionally narrow:
`PUT /admin/v1/config` accepts `target_base_url` updates only and rejects malformed provider URLs.

Request/session telemetry records privacy-safe metadata only: request ID, session ID, provider,
method, route, query presence, status, latency, error kind, start time, and token counts when known.
Prompt bodies, tool output, provider credentials, and raw request or response payloads are not stored
by default.

The initial metrics and diagnostics endpoints summarize request totals, error totals, average
latency, status counts, recent failure categories, health, paths, and proxy/admin listen addresses
from the privacy-safe telemetry store.

Telemetry retention defaults to 1,000 recent metadata records or 24 hours, whichever removes more.
Cleanup supports dry-run planning before confirmed removal. Cachy does not yet persist structured
logs itself, so these retention controls currently apply to request/session metadata; persistent log
retention belongs with the future storage backend.

For v1, the Electron app can invoke CLI commands directly and read JSON output. Move to the admin API
once the CLI behavior is stable.

## Suggested UI

Primary views:

- Dashboard: proxy status, active provider, savings today, recent errors.
- Sessions: recent requests grouped by agent/app, with before/after token counts.
- Providers: OpenAI, Anthropic, OpenRouter, Ollama, LM Studio, vLLM, custom endpoint.
- Integrations: Codex, Claude, MCP, shell environment.
- Storage: CCR cache size, retention, cleanup.
- Diagnostics: ports, binary version, config path, network checks.

Avoid making this a marketing page. The first screen should be the operational dashboard.

## Packaging

Use Electron for desktop distribution and bundle the Go binary inside the app:

```text
resources/
  bin/
    win32-x64/cachy.exe
    darwin-arm64/cachy
    darwin-x64/cachy
    linux-x64/cachy
```

The standalone CLI should remain independently downloadable. The app should discover an existing
installed CLI before falling back to the bundled binary.

Recommended stack:

- Electron + TypeScript.
- Vite for renderer build.
- React for UI.
- electron-builder or Electron Forge for packaging.
- `lucide-react` for icons.
- `zustand` or TanStack Query for app state.

## Security Notes

- Bind admin APIs to `127.0.0.1` only.
- Require a per-run or per-install admin token.
- Never expose provider API keys to renderer code directly.
- Store secrets through OS keychain where possible.
- Keep the proxy functional without Electron running.
- Treat Electron as a controller, not as the trusted request path.

## MVP Cut

1. Bundle or locate `cachy`.
2. Start/stop proxy.
3. Show `/healthz` and version.
4. Edit target provider URL.
5. Install Codex and Claude integrations by invoking CLI commands.
6. Show recent structured logs.

That gives users a useful app without coupling the product to Electron too early.

## Implementation Status

The initial Electron, Vite, React, and TypeScript scaffold lives under `desktop/`. It provides a
minimal operational shell and scripts for development, tests, typechecking, and production renderer
builds.

The main process now owns Cachy binary discovery and process supervision. It searches installed
`cachy` binaries on `PATH` before falling back to the bundled `resources/bin/<platform>-<arch>`
layout, starts the proxy with explicit listen and target arguments, and exposes start, stop, and
status operations through preload IPC. Admin API status polling stays in the main process so renderer
code does not receive provider keys or admin tokens.

Dashboard expansion, provider forms, integration screens, and detailed diagnostics remain tracked in
later desktop companion issues.

The dashboard renderer now consumes preload IPC status and renders loading, empty, healthy, error,
disabled, and permission-denied states. It shows current proxy state, provider target, savings
placeholder, recent failure count, and the next useful start/stop or recovery action.

Provider, integration, and diagnostics panels are now present in the renderer. Provider target saves
validate HTTP(S) URLs before crossing IPC. Integration panels request Cachy dry-run command output.
Diagnostics panels render admin health, listen address, and recent failure categories while surfacing
admin API failures as user-visible operational state.

Renderer-facing provider config is sanitized at the Electron main-process boundary and defensively
normalized in the renderer. Raw provider keys and admin tokens must not appear in IPC payloads,
status messages, or error text.
