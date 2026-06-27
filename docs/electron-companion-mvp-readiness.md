# Electron Companion MVP Readiness

## Status

Ready for the current MVP scope.

## Scope

The Electron companion MVP covers the optional desktop control surface layered over the Go runtime:

- Electron, Vite, React, and TypeScript scaffold under `desktop/`.
- Main-process Cachy binary discovery that prefers an installed `cachy` on `PATH` and falls back to
  the bundled `resources/bin/<platform>-<arch>` layout.
- Start, stop, restart, and status IPC for the supervised Go proxy.
- Admin API status, config, and diagnostics reads from the Electron main process.
- Provider target editing with renderer-side URL validation and Go-admin validation.
- Codex, Claude, and MCP integration dry-run command output from the desktop app.
- Operational dashboard states for loading, empty, healthy, error, disabled, and permission-denied
  paths.
- Secret-boundary tests that prevent raw provider credentials or admin tokens from crossing into
  renderer-facing payloads or status text.

## Validation Evidence

- Issue #34 closed after the Electron, Vite, React, and TypeScript scaffold landed.
- Issue #35 closed after binary discovery, process supervision, and status/start/stop IPC landed.
- Issue #36 closed after the dashboard showed proxy status, provider target, recent failures,
  savings placeholder, and next action states.
- Issue #37 closed after provider, integration, and diagnostics views landed.
- Issue #38 closed after Electron secret-boundary and admin API failure tests landed.
- Issue #10 closed for the admin and observability baseline used by the companion app.
- ADR 0004 records the accepted boundary: Electron is a companion controller, while Go owns the
  trusted request path, provider forwarding, persistence, secrets, and admin/config validation.

Before closing this readiness item, run:

```text
go test ./...
go vet ./...
npm audit --audit-level=moderate
npm run test
npm run typecheck
npm run build
git diff --check
```

Run the npm commands from `desktop/`.

## Supported Behavior

The desktop app can operate as a local companion without replacing the Go runtime. The Electron main
process discovers or supervises the Go binary and communicates with the local admin API. Renderer code
receives only bounded operational state through preload IPC.

The dashboard makes the current proxy state visible and keeps useful recovery actions close to the
status that needs them. Provider settings expose target URL editing and credential presence only; raw
provider keys do not appear in renderer payloads. Integration views surface dry-run command output so
users can inspect Codex, Claude, and MCP setup actions before mutation. Diagnostics expose health,
listen addresses, and recent failure categories from the admin API.

## Known Boundaries

- Electron is optional; Cachy must remain usable from the Go CLI and proxy without the desktop app.
- Electron does not proxy model traffic, own compression, write local persistence directly, or
  validate security-sensitive config outside the Go admin surface.
- Packaged installers, code signing, auto-update, and bundled binary release assembly are release
  packaging work.
- Live compression savings, session history, CCR storage management, and richer local activity
  timelines remain future UI work.
- OS keychain storage for provider credentials is not part of this MVP; the current app exposes only
  credential presence.
