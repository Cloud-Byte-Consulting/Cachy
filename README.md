# Cachy

Cachy is a clean-room Go-first LLM context optimization proxy.

Current status: the transparent proxy, CLI/platform foundation,
admin/observability baseline, cache-safe native compression foundation, local
CCR storage foundation, agent integration MVP, Electron companion MVP, WASM
plugin host MVP, and release packaging MVP are
implemented and validated. Cachy can proxy OpenAI-compatible and Anthropic-compatible HTTP
traffic to a configured upstream while preserving provider-visible request and
response semantics. SSE responses are streamed through without buffering the
full response, large bodies pass through, upstream provider failures are
preserved, and unreachable upstreams return `502 Bad Gateway`.

The next implementation phase is local content retrieval.

> **Scope note:** The live proxy request path is currently transparent
> pass-through only (`internal/proxy/upstream.go` forwards bodies via `io.Copy`
> and does not transform provider payloads, as recorded in
> [Transparent proxy MVP readiness](docs/transparent-proxy-mvp.md)). The
> compression, CCR, and WASM plugin items below are implemented and validated as
> standalone library/CLI foundations under `internal/` and `pkg/`, but they are
> **not yet wired into the live proxy** request/response flow. Admin and
> telemetry surfaces are active; only request-metadata recording runs in the
> proxy path, not payload transformation. These foundations remain on the
> roadmap for activation.

## Current Capabilities

- Health endpoint at `/healthz`.
- OpenAI-compatible chat completions pass-through.
- OpenAI-compatible responses API pass-through.
- Anthropic messages API pass-through.
- Provider-safe header forwarding for auth, version, content, and metadata
  headers.
- Hop-by-hop, proxy provenance, and transport-managed request headers stripped
  before upstream forwarding.
- SSE pass-through with flush behavior covered by live proxy tests.
- Large request and response body pass-through tests.
- Client cancellation propagation to the upstream request context.
- Unified `cachy` command surface with `proxy` and `doctor` commands.
- Platform-aware config, state, and cache path resolution for Windows, macOS,
  and Linux.
- Privacy-safe structured proxy logging with secret and prompt redaction.
- `cachy doctor` diagnostics for version, platform paths, target URL, listen
  port, provider reachability, credential presence, and Codex/Claude/MCP
  integration state without printing secret values.
- Localhost-only admin token guard for private control APIs.
- Admin status/config endpoints with redacted provider credential presence and
  narrow target URL updates.
- Privacy-safe request/session telemetry for request IDs, session IDs,
  provider, route, latency, status, error kind, and token counts when known.
- Admin metrics and diagnostics endpoints for request totals, error totals,
  latency, status counts, failure categories, health, paths, and listen
  addresses.
- Metadata retention controls with dry-run and confirmed cleanup paths.
- Cache-safe live-zone detection for OpenAI chat and Anthropic messages.
- Token counting with exact provider/model counter registration and a
  deterministic estimator fallback.
- Native compression proposal pipeline with UTF-8 validation, token-savings
  validation, and original-content fallback.
- Native compressors for text, logs, JSON, diffs, and code-like live-zone
  content.
- Integration coverage proving stable cache hot zones remain unchanged while
  selected tool-result live blocks compress.
- CCR marker parsing/rendering with SHA-256 content addresses.
- Local content-addressed CCR store rooted under the platform state directory.
- CCR retention cleanup, size diagnostics, recovery, and privacy tests.
- Reversible Codex and Claude Code integration install, repair, uninstall, and
  dry-run command support.
- MCP registration snippets and OpenAI-compatible endpoint recipes for Ollama,
  llama.cpp server, LM Studio, vLLM, LocalAI, and custom targets.
- Integration doctor guidance with dry-run install or repair commands for
  missing and incomplete agent setup.
- WASM plugin manifest validation for v1 compressor plugins with bounded
  timeout, memory, input, and output limits.
- Default-deny wazero host that gives plugins only selected live-zone blocks and
  Cachy host ABI functions.
- Plugin lifecycle CLI commands for list, inspect, enable, disable, and fixture
  testing.
- Malicious, slow, oversized, invalid, and protected-field plugin output tests.
- Native-first compression pipeline ordering that can try enabled WASM
  compressor plugins after native proposals, with the same UTF-8 and
  token-savings validation before provider-visible content changes.
- Release build workflow for six CLI targets, platform archives with checksums,
  multi-arch Docker OCI artifact generation, SBOM/release-note evidence, and
  install smoke tests.
- CI gates for formatting, vet, unit/integration tests, race tests, and
  cross-compilation for the six required CLI targets.
- Public composable Go packages under `pkg/` for embedding the proxy,
  compression pipeline, token counting, CCR primitives, platform paths, and
  observability helpers in other Go programs.

## Development

Run the proxy through the unified CLI:

```powershell
go run ./cmd/cachy proxy --listen 127.0.0.1:8787 --target http://127.0.0.1:11434
```

The target can also come from the environment:

```powershell
$env:CACHY_TARGET_BASE_URL = 'http://127.0.0.1:11434'
go run ./cmd/cachy proxy --listen 127.0.0.1:8787
```

Health check:

```powershell
curl http://127.0.0.1:8787/healthz
```

Run tests:

```powershell
go test ./...
go vet ./...
```

Run the Go suite with Ginkgo/Gomega and emit JUnit for pipelines:

```powershell
.\scripts\test-go-junit.ps1
```

```bash
bash ./scripts/test-go-junit.sh
```

The JUnit report is written to `reports/go/go-junit.xml`.

Run local diagnostics:

```powershell
go run ./cmd/cachy doctor --target http://127.0.0.1:11434
```

Build the CLI for the local platform:

```powershell
go build ./cmd/cachy
```

Run the Electron companion scaffold:

```powershell
cd desktop
npm ci
npm run dev
```

Validate the desktop package:

```powershell
cd desktop
npm run test
npm run typecheck
npm run build
```

CI verifies the supported Windows, macOS, and Linux AMD64/ARM64 targets listed
in [Cross-platform support](docs/cross-platform-support.md) and runs the
desktop package audit, tests, typecheck, and build.

## Admin And Observability

The admin API building blocks are implemented in `internal/admin` for companion
apps and future CLI workflows. Admin handlers must bind to loopback addresses
and require either `Authorization: Bearer <token>` or `X-Cachy-Admin-Token`.

Implemented local control surfaces:

```text
GET /admin/v1/status
GET /admin/v1/config
PUT /admin/v1/config
GET /admin/v1/metrics
GET /admin/v1/diagnostics
```

Telemetry stores metadata only by default. Prompt bodies, tool output, provider
credentials, and raw request/response payloads are not retained.

## Documentation

- [Architecture options](docs/architecture-options.md) records the clean-room
  rewrite plan and phase order.
- [Transparent proxy MVP readiness](docs/transparent-proxy-mvp.md) records the
  current supported pass-through behavior, validation evidence, and known
  limitations.
- [Cache-safe compression MVP readiness](docs/compression-mvp-readiness.md)
  records the native compression behavior, validation evidence, and known
  limitations.
- [CCR architecture](docs/ccr-architecture.md) records the local retrieval
  marker format, hash strategy, compatibility rules, and privacy boundary.
- [Agent integrations](docs/agent-integrations.md) records Codex, Claude Code,
  MCP, endpoint recipe, and doctor setup flows.
- [Agent integration MVP readiness](docs/agent-integration-mvp-readiness.md)
  records the validation evidence, supported behavior, and known boundaries for
  the current integration wave.
- [Electron companion MVP readiness](docs/electron-companion-mvp-readiness.md)
  records the validation evidence, supported behavior, and known boundaries for
  the current desktop companion wave.
- [WASM plugin host MVP readiness](docs/wasm-plugin-mvp-readiness.md) records
  the validation evidence, supported behavior, and security boundaries for
  opt-in plugin use.
- [Release packaging MVP readiness](docs/release-packaging-mvp-readiness.md)
  records the validation evidence, supported behavior, and known boundaries for
  release candidate packaging.
- [Composable Go API](docs/composable-go-api.md) records the public package
  surface for embedding Cachy logic in other Go programs.
- [Cross-platform support](docs/cross-platform-support.md) records target
  platforms, packaging expectations, and compatibility tests.
- [ADRs](docs/adr) record accepted project decisions.
