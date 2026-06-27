# Transparent Proxy MVP Readiness

This note records the current transparent proxy behavior after the proxy foundation work. It is
limited to pass-through behavior; compression, CCR, Electron, WASM plugins, and agent installers
remain separate phases.

## Supported Pass-Through Behavior

- `/healthz` returns `200 OK` with `ok`.
- Requests are forwarded to the configured target base URL while preserving method, path, query,
  body, and provider-visible end-to-end headers.
- OpenAI-compatible chat completions requests pass through with provider-visible request and
  response semantics preserved.
- OpenAI-compatible responses API requests pass through with provider-visible request and response
  semantics preserved.
- Anthropic messages requests pass through with version, beta, API key, content headers, status,
  response headers, and body semantics preserved.
- SSE responses with `Content-Type: text/event-stream` are copied without buffering the complete
  response. Events are flushed to the client as the upstream emits them.
- Large request and response bodies pass through without application-level reserialization.
- Upstream provider error responses preserve upstream status, headers, and body.
- Unreachable upstreams return `502 Bad Gateway`.
- Client cancellation is propagated to the upstream request context.

## Header And Auth Policy

The proxy forwards provider auth and provider metadata headers such as `Authorization`, `X-Api-Key`,
`Anthropic-Version`, `Anthropic-Beta`, `OpenAI-Organization`, and `OpenAI-Project`.

The proxy strips hop-by-hop headers, proxy provenance headers, and request transport headers before
forwarding upstream. This includes `Connection`, `Proxy-Authorization`, `Forwarded`,
`X-Forwarded-For`, `X-Real-IP`, `Via`, and `Content-Length`.

Credential-like headers are covered by privacy-safe structured logging and telemetry helpers. The
proxy does not store prompt bodies, provider credentials, or raw request/response payloads in
telemetry by default.

## Validation Evidence

The transparent proxy fixture suite covers:

- Current scaffold and health behavior.
- Provider-safe header forwarding and redaction helpers.
- OpenAI-compatible chat completions and responses API fixtures.
- Anthropic messages API fixtures.
- SSE flush behavior with a live proxy server.
- Large bodies, cancellation propagation, upstream provider failures, and unreachable upstreams.

Before closing readiness, the following local commands passed:

```text
go test ./internal/proxy
go test ./...
go vet ./...
gofmt -l $(git ls-files '*.go')
```

CI also runs formatting, vet, tests, race tests, and cross-compiles the six required CLI targets on
every pull request and push to `main`.

## Known Limitations

- The proxy is transparent pass-through only; it does not yet compress, cache, or rewrite provider
  payloads.
- Provider routing is configured as a single target base URL. Multi-provider routing and explicit
  provider adapters are future work.
- WebSocket pass-through is not implemented yet.
- CLI/platform and admin/observability baselines are tracked outside this proxy-specific note and
  are complete for the current foundation.
- Release packaging, Docker images, SBOMs, installers, and install smoke tests are future phases.
