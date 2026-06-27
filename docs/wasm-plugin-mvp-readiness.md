# WASM Plugin Host MVP Readiness

## Status

Ready for the current MVP scope.

## Scope

The WASM plugin host MVP covers the opt-in plugin extension lane for selected live-zone compression:

- Manifest loading and validation for v1 plugins with explicit compressor capabilities.
- Bounded plugin limits for timeout, memory, input size, and output size.
- Default-deny `wazero` execution without WASI, filesystem, networking, environment, or arbitrary
  host imports.
- Selected-block host envelope that sends plugins live-zone block metadata and text, not the full
  provider request.
- Strict plugin output validation for UTF-8, JSON schema, size, and allowed actions.
- Plugin lifecycle CLI commands for list, inspect, enable, disable, and fixture testing.
- WASM compressor adapter that fits behind the native compression proposal contract after native
  compressors.

## Validation Evidence

- Issue #40 closed after the manifest schema, fixture validation, capability allowlist, and limit
  checks landed.
- Issue #41 closed after the default-deny `wazero` host landed.
- Issue #42 closed after plugin lifecycle CLI commands landed.
- Issue #43 closed after malicious, slow, oversized, invalid-output, invalid-UTF-8, trap, and
  protected-field mutation tests landed.
- Issue #44 closed after the WASM compressor adapter and native-first ordered compression pipeline
  landed.
- Issue #10 closed for the admin and observability baseline.
- Issue #16 closed for the cache-safe compression validation pipeline used by WASM proposals.
- Issue #23 closed for the CCR storage contract and marker ownership boundary.
- ADR 0005 records the accepted sequencing: WASM plugins are deferred until the transparent proxy,
  native compression, CCR storage, token validation, and admin/status API are stable.

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

Plugins are optional and disabled unless their lifecycle marker is present. Cachy reads plugin
directories from the lifecycle path, validates each manifest, reads `plugin.wasm`, and only runs
enabled plugins whose `compress` capability matches the selected live-zone block content type.

The Go runtime remains the host and decision-maker. Native compressors run first. If a native
proposal does not pass validation, an enabled WASM compressor can propose a replacement through the
same compression contract. Cachy applies that replacement only after the existing UTF-8 and
token-savings checks accept it. `keep`, `annotate`, and `reject` plugin actions fall back to the
original block text.

The plugin host exposes only the Cachy input and output ABI, enforces manifest limits, rejects
unexpected imports, and treats every plugin result as untrusted. Plugins do not receive provider API
keys, full prompt history, filesystem access, network access, CCR write authority, or direct control
over provider-visible protected fields.

## Known Boundaries

- WASM plugins are not used for HTTP proxying, SSE/WebSocket streaming, provider auth, token
  accounting, CCR storage, or admin API behavior.
- Plugins are not loaded into the transparent proxy request path by default; runtime configuration
  and product UX remain later work.
- The current plugin contract covers compressor proposals only. Redaction, classification, policy
  modules, richer metadata outputs, and plugin packaging/distribution require later explicit
  decisions.
- Go controls CCR marker policy and storage. Plugins can propose text but cannot write retrieval
  records or decide retention.
- The Electron app can manage plugin directories later, but the Go CLI remains the source of truth
  for current lifecycle operations.
