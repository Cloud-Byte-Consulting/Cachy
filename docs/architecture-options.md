# Cachy Architecture Options

This repo starts as a clean-room replacement plan for an LLM context/cache optimization proxy.
Do not copy source files, comments, README text, images, package metadata, or test fixtures from
Headroom unless the project intentionally becomes an Apache-2.0 derivative and preserves the
required notices.

## Source Reference Boundary

The Headroom repository, including
`http://truenas-scale-1.tail5a208d.ts.net:30008/Cloud-Byte-Consulting/headroom.git`, is reference
material only. Cachy implementation work must follow the decisions recorded in this repository's
docs and ADRs. When a needed product, architecture, security, licensing, or packaging decision is
not already covered here, pause and ask the project owner before choosing a new direction.

## Goals

- Provide an OpenAI/Anthropic-compatible proxy for context reduction and cache preservation.
- Replace Rust request-path components with Go.
- Minimize or remove Python from runtime installs.
- Keep agent integrations practical for Codex, Claude Code, and local OpenAI-compatible servers.
- Preserve provider cache hot zones by avoiding mutation of stable prompt prefixes.

## Explicit Non-Goals

- Do not bundle, install, or depend on Serena.
- Do not require a semantic code server for core compression.
- Do not make MCP setup install unrelated companion tools by default.
- Do not copy Headroom's install behavior where it configures optional external tools as part of
  the primary wrapper flow.

## Legal And Branding Guardrails

There are two viable paths:

1. Clean-room rewrite.
   - Use Headroom only as behavioral inspiration.
   - Write new code, new docs, new tests, new names, new package metadata, and new assets.
   - Do not copy implementation details line-for-line.
   - Cachy can choose its own license.

2. Derivative fork.
   - Keep Apache-2.0 license terms.
   - Preserve attribution and required notices.
   - Clearly document modifications.
   - Branding can change, but license and notice obligations remain.

For commercial or closed-source distribution, prefer clean-room rewrite with a short written design
spec and independent implementation notes.

## Proposed Go Architecture

```text
cmd/
  cachy/
    main.go                 # CLI entrypoint
  cachy-proxy/
    main.go                 # HTTP/SSE/WebSocket proxy

internal/
  proxy/
    server.go               # route setup, health, middleware
    upstream.go             # provider forwarding
    headers.go              # provider-safe header filtering
    authmode.go             # API-key/OAuth/subscription policy classification
  providers/
    openai.go               # /v1/chat/completions, /v1/responses
    anthropic.go            # /v1/messages
    openrouter.go           # OpenAI-compatible variant
    bedrock.go              # later, SigV4
    vertex.go               # later, ADC
  stream/
    sse.go                  # byte-safe SSE parser/writer
    websocket.go            # Codex-style WebSocket pass-through
  compress/
    pipeline.go             # live-zone orchestration
    detector.go             # content type detection
    json.go                 # JSON summarization/minification
    logs.go                 # log compaction
    code.go                 # code-aware selection
    diff.go                 # diff compaction
    text.go                 # plain text truncation/summarization
    validate.go             # token-count validation and fallback
  ccr/
    store.go                # content-addressed retrieval store
    sqlite.go               # local persistent backend
    marker.go               # retrieval marker format
  tokens/
    counter.go              # tokenizer interface
    tiktoken.go             # OpenAI-ish token counting
    estimate.go             # fallback estimator
  install/
    claude.go               # Claude config/wrapper install
    codex.go                # Codex config/wrapper install
    mcp.go                  # MCP registration
  config/
    config.go               # env + file config
  observability/
    metrics.go              # Prometheus/OpenTelemetry hooks
    savings.go              # token/cost accounting

pkg/
  sdk/
    client.go               # optional public Go SDK
```

## Runtime Shape

The smallest useful product is a single Go binary:

```text
cachy proxy --listen 127.0.0.1:8787 --target http://127.0.0.1:11434
```

Clients point at it with:

```text
OPENAI_BASE_URL=http://127.0.0.1:8787/v1
ANTHROPIC_BASE_URL=http://127.0.0.1:8787
```

The proxy should initially support:

- OpenAI-compatible chat completions.
- OpenAI responses API pass-through with conservative compression.
- Anthropic messages API.
- Streaming SSE pass-through.
- Local OpenAI-compatible backends such as Ollama, llama.cpp server, LM Studio, vLLM, and LocalAI.

Current transparent proxy readiness notes, validation evidence, and known limitations are recorded
in [Transparent Proxy MVP Readiness](transparent-proxy-mvp.md).

Current CLI/platform readiness notes, supported targets, and platform path expectations are recorded
in [Cross-Platform Support](cross-platform-support.md).

## Live-Zone Compression Model

Cachy treats provider cache hot zones as protected by default. The initial live-zone detector models
OpenAI chat messages and Anthropic messages, but selects only tool-result content as eligible for
compression. System, developer, ordinary user, and assistant message content remains stable and is
not selected by default. This conservative rule keeps stable prompt prefixes intact until a later
decision broadens eligible request regions.

Detected live blocks include provider, JSON path, role, source, stability, content type, text, and
selection metadata. The first content classifier recognizes JSON, diff, fenced/code-like text, log
output, and plain text so later compressor PRs can route blocks without reparsing full requests.

Token counting is behind an internal provider/model counter interface. Exact counters can be
registered when a tokenizer is available for a provider/model pair; otherwise Cachy uses a
deterministic conservative estimator. Compression validation compares original and proposed block
counts and treats a proposal as useful only when the counted or estimated token delta is positive.

The native compression pipeline treats every compressor result as an untrusted proposal. It invokes
compressors only for selected live blocks, preserves stable/protected blocks unchanged, rejects
invalid UTF-8 output, and falls back to the original block whenever token validation shows no
positive savings. Later native and WASM compressors fit behind this block-in/proposal-out contract.

The first native compressors handle plain text and log/tool-output content conservatively by
collapsing adjacent repeated paragraphs or log lines while keeping the first occurrence and nearby
diagnostic signal. They deliberately leave JSON, diff, and code blocks unchanged for later
specialized compressors.

Structured native compressors handle JSON, diff, and code-like live blocks separately. JSON is
compacted only when it parses successfully. Diff compression keeps file headers, hunk headers, and
added/removed lines while omitting unchanged context. Code compression collapses adjacent repeated
lines and repeated blank lines, which targets generated-code noise without parsing language syntax.

## Python Replacement Options

### Option A: Go-Only Runtime

Best default for Cachy.

- Go implements proxy, CLI, installers, compression pipeline, CCR store, metrics, and provider routing.
- Python is used only for development scripts if needed.
- Distribution is simple: one binary per platform, Docker image optional.
- Hardest parts are tokenization parity and ML/OCR-style compression.

Recommended when the goal is easy Windows install, low dependency count, and operational simplicity.

### Option B: Go Proxy Plus TypeScript SDK

Strong if the product targets app developers.

- Go owns proxy and CLI.
- TypeScript provides Vercel AI SDK, OpenAI SDK, Anthropic SDK, and browser/server adapters.
- No Python runtime for users.
- Adds npm packaging work but improves adoption in JS-heavy LLM apps.

Recommended if Cachy should be both a proxy and an application SDK.

### Option C: Go Runtime Plus Python Plugin Boundary

Useful only if Python ML features are must-have early.

- Go runs the request path.
- Python workers are optional sidecars for OCR, embeddings, reranking, or research compressors.
- Go calls Python over localhost HTTP/gRPC, never via in-process bindings.
- Users who do not enable ML do not install Python.

Recommended only for advanced compression features that do not have good Go equivalents yet.

### Option D: TypeScript/Node Runtime Instead Of Python

Good for integrations, weaker for proxy performance.

- Node replaces Python for CLI/installers/SDK.
- Go still runs the proxy, or Node runs a lighter proxy.
- Very friendly to AI app developers, but shipping a global CLI brings npm/node version concerns.

Recommended for SDK-first packaging, not for the core proxy.

### Option E: Keep Python For CLI Only

Least disruptive, but not ideal for the stated goal.

- Go replaces Rust and request-path proxy.
- Python remains for wrappers/installers.
- Still exposes users to Python install problems on Windows.

Recommended only as a temporary migration bridge.

## Recommended Path

Use Option B:

- Go-only runtime and CLI.
- Optional TypeScript SDK for app/framework integrations.
- No Python dependency for end users.
- Optional ML sidecars later if a feature justifies the weight.

## Rewrite Phases

1. Foundation
   - Create Go module, CLI skeleton, config loader, health endpoint, provider target config.
   - Add CI for Windows, Linux, and macOS builds.
   - Status: complete for the current proxy, CLI command surface, platform path resolver, doctor
     diagnostics, privacy-safe logging, ADR baseline, and CI quality gates.

2. Transparent Proxy
   - Implement OpenAI-compatible and Anthropic-compatible pass-through.
   - Preserve request bytes as much as possible.
   - Add SSE pass-through tests.
   - Status: complete for the current transparent pass-through baseline.

3. Cache-Safe Compression
   - Implement live-zone-only block selection.
   - Add JSON, log, diff, code, and text compressors.
   - Validate token savings before mutation.
   - Status: complete for the current opt-in native compression foundation. See
     [Cache-Safe Compression MVP Readiness](compression-mvp-readiness.md) for supported behavior,
     validation evidence, and known limitations.

4. CCR
   - Store originals locally by content hash.
   - Inject retrieval markers/tools only when compatible with the provider/client.
   - Marker contract: v1 markers use SHA-256 content addresses, include only the original byte
     count, and are rendered/parsed by Go. See [CCR Architecture](ccr-architecture.md).
   - Status: complete for the current local CCR storage foundation. Proxy-path marker injection
     remains future integration work.

5. Agent Installers
   - Codex and Claude setup commands.
   - Windows-safe behavior without Unix hook assumptions.
   - Status: complete for the current agent integration MVP. Codex and Claude Code install, repair,
     uninstall, dry-run support, MCP registration snippets, endpoint recipes, and integration doctor
     diagnostics are implemented. See [Agent Integrations](agent-integrations.md) and
     [Agent Integration MVP Readiness](agent-integration-mvp-readiness.md).

6. Observability
   - Token accounting, compression ratios, provider cache usage, latency.
   - Prometheus endpoint and structured logs.
   - Status: baseline complete for privacy-safe structured logs, request/session metadata,
     admin status/config, metrics, diagnostics, and in-memory metadata retention controls.
     Compression-specific savings and provider cache signals remain part of later compression work.

7. SDK And Local LLM Polish
   - TypeScript adapters.
   - Ollama/LM Studio/vLLM recipes.
   - Conservative defaults for local models.

8. Electron Companion
   - Optional desktop controller for local status, provider setup, integrations, and diagnostics.
   - Status: complete for the current companion MVP. Electron remains outside the trusted request
     path while the Go runtime owns proxying, persistence, secrets, and admin/config validation. See
     [Electron Companion MVP Readiness](electron-companion-mvp-readiness.md).

## Initial Go Dependency Candidates

- HTTP: standard library `net/http` first; consider `chi` only if routing grows.
- CLI: `spf13/cobra` or `urfave/cli/v3`.
- Config: `spf13/viper` or a small env/file loader.
- Logging: `log/slog`.
- Metrics: `prometheus/client_golang`.
- SQLite: `modernc.org/sqlite` for CGO-free builds, or `mattn/go-sqlite3` if CGO is acceptable.
- Tokenization: evaluate `tiktoken-go`; keep estimator fallback.
- JSON byte preservation: standard `encoding/json` is not byte-preserving; use raw byte ranges and `json.RawMessage`.
- SSE: small internal parser is reasonable.
- WebSocket: `nhooyr.io/websocket` or `gorilla/websocket`.
- AWS: official AWS SDK for Go v2.
- GCP: official Google auth libraries.

## Python Features To Drop Or Defer

- Heavy ML text compression.
- OCR/image compression.
- LiteLLM provider abstraction.
- In-process embedding memory.
- Broad framework integrations.

These can come back later as optional sidecars or SDK features.
