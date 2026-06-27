# Cache-Safe Compression MVP Readiness

This note records the current cache-safe compression behavior after the native compression wave.
Compression is ready for opt-in internal use by future proxy integration work. The transparent
proxy still remains pass-through by default until a later issue wires request rewriting into the
runtime path.

## Supported Compression Behavior

- OpenAI chat and Anthropic messages requests can be inspected for live-zone blocks.
- Stable cache hot zones are protected by default. System, developer, ordinary user, and assistant
  message content is not selected for compression.
- Tool-result content is selected as the first live-zone compression target.
- Detected live blocks include provider, JSON path, role, source, stability, content type, text, and
  selection metadata.
- Compression output is treated as an untrusted proposal. Cachy applies a proposal only when it is
  valid UTF-8 and token validation reports positive savings.
- Original block text is preserved when a block is protected, a compressor returns invalid output, a
  compressor cannot reduce the block, or token validation is inconclusive.
- Exact token counters can be registered per provider/model. When no exact counter is available,
  Cachy uses the deterministic estimator fallback.

## Native Compressors

Current native compressors cover:

- Plain text: collapses adjacent repeated paragraphs.
- Logs and tool output: collapses adjacent repeated log lines while preserving diagnostic lines.
- JSON: compacts valid JSON and leaves malformed JSON unchanged for validation fallback.
- Diffs: preserves file headers, hunk headers, and added/removed lines while omitting unchanged
  context.
- Code-like content: collapses adjacent repeated lines and repeated blank lines, targeting
  generated-code noise without parsing language syntax.

## Validation Evidence

The compression fixture suite covers:

- OpenAI and Anthropic live-zone detection.
- Content classification for text, log, code, diff, and JSON.
- Token counting with exact-counter registration and estimator fallback.
- Proposal validation for positive savings, no savings, larger output, invalid UTF-8, and protected
  blocks.
- Native text, log, JSON, diff, and code compressor behavior.
- Integration coverage proving OpenAI and Anthropic stable hot zones remain unchanged while selected
  tool-result live blocks compress.

Before closing readiness, the following local commands passed:

```text
go test ./internal/compress
go test ./...
go vet ./...
git diff --check
```

CI also runs formatting, vet, tests, race tests, and cross-compiles the six required CLI targets on
every pull request and push to `main`.

## Known Limitations

- Compression is not enabled in the transparent proxy request path yet.
- OpenAI responses API compression remains conservative future work.
- Only tool-result content is selected by default; broad user-message or assistant-message
  compression requires a separate product decision.
- Native compressors are deterministic and conservative; semantic summarization, embeddings,
  OCR/image compression, and ML compression are deferred.
- CCR storage and retrieval markers are separate future work.
- WASM plugins and Electron UI surfaces are out of scope for the compression MVP.
