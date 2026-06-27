# WASM Plugin Architecture

WASM is useful for Cachy as an optional sandboxed plugin layer. It should not be the core runtime.
The Go proxy remains responsible for HTTP, streaming, provider routing, persistence, security, and
compression orchestration.

## Recommendation

Implement WASM after the native Go compression pipeline exists.

Use it for:

- Third-party compressors.
- Enterprise policy modules.
- Custom redaction/sanitization.
- Content classifiers.
- Lightweight scoring rules.
- Customer-specific routing decisions.

Do not use it for:

- HTTP proxying.
- SSE/WebSocket streaming.
- SQLite/CCR storage.
- Provider auth.
- Token accounting.
- First-party v1 compressors.

## Runtime Shape

```text
Client
  │
  ▼
Go proxy
  ├── provider protocol handling
  ├── cache-safe live-zone selection
  ├── token counting
  ├── CCR storage
  ├── built-in Go compressors
  └── plugin host
        ├── load plugin manifest
        ├── validate permissions
        ├── run WASM module
        ├── enforce timeout/memory limits
        └── validate result before applying
  │
  ▼
Provider / local LLM
```

The WASM module never sees the whole request by default. The Go host gives it only the selected
live-zone block plus a small metadata object.

The initial host is a default-deny `wazero` runner. It does not instantiate WASI, filesystem,
network, environment, or arbitrary host modules. Plugins can import only the Cachy host functions
listed below, and the host enforces the manifest timeout, memory, input, and output limits.

## Plugin Contract

The host sends a JSON envelope:

```json
{
  "api_version": "v1",
  "request_id": "req_123",
  "provider": "openai",
  "model": "gpt-4.1",
  "content_type": "log",
  "token_budget": 1200,
  "block": {
    "id": "block_1",
    "role": "tool",
    "text": "..."
  },
  "metadata": {
    "is_error": false,
    "source": "tool_result"
  }
}
```

The v1 host ABI exposes this envelope through a tiny imported module named `cachy`:

```text
cachy.input_len() -> i32
cachy.input_read(ptr: i32) -> i32
cachy.output_write(ptr: i32, len: i32) -> i32
```

Plugins export `run() -> i32`, read the selected-block envelope from `input_read`, and write a JSON
result through `output_write`. A plugin that imports WASI, filesystem, networking, environment, or
other host functions fails to instantiate.

The plugin returns:

```json
{
  "action": "replace",
  "text": "...",
  "summary": "trimmed repeated stack frames",
  "confidence": 0.92,
  "lossiness": "lossy"
}
```

The host rejects malformed output before the compression pipeline can apply it. Output must be valid
UTF-8, valid JSON, within `max_output_bytes`, and limited to the v1 result schema fields. Unknown
fields are rejected so plugins cannot smuggle protected-field mutation requests into the result.

Allowed actions:

- `keep`: no change.
- `replace`: replace the selected block text.
- `annotate`: return metadata only; Go decides what to do.
- `reject`: plugin declines the block.

The Go host always validates:

- Output is valid UTF-8.
- Output is below configured size limits.
- Token count improves when compression is required.
- Provider cache hot zone is untouched.
- Protected fields are untouched.
- CCR marker rules are followed by Go, not by the plugin.

## Plugin Manifest

Each plugin ships a manifest next to its `.wasm` file:

```toml
name = "stacktrace-compressor"
version = "0.1.0"
api_version = "v1"
description = "Compresses repeated stack frames and dependency noise."

[capabilities]
compress = ["log", "text"]
classify = []
redact = []

[limits]
timeout_ms = 50
memory_mb = 32
max_input_bytes = 262144
max_output_bytes = 131072
```

The initial v1 manifest validator accepts:

- `api_version = "v1"` only.
- Lowercase plugin names made from letters, numbers, dots, underscores, and dashes.
- Semantic-version-like `version` values.
- `compress` capabilities for `text`, `log`, `json`, `diff`, and `code` live blocks.
- Empty `classify` and `redact` capability lists. Non-empty lists are reserved for later explicit
  capability decisions.
- `timeout_ms` from 1 to 1,000.
- `memory_mb` from 1 to 128.
- `max_input_bytes` from 1 byte to 1 MiB.
- `max_output_bytes` from 1 byte to 512 KiB, and no larger than `max_input_bytes`.

Manifest parsing uses TOML so plugin packages can keep a small human-readable contract next to the
`.wasm` module. The host still treats the decoded manifest as untrusted input and validates every
field before enabling a plugin.

## Go Host Interface

Internally, WASM plugins should fit behind the same interface as native Go compressors:

```go
type Compressor interface {
    Name() string
    CanHandle(block Block) bool
    Compress(ctx context.Context, block Block, budget Budget) (Result, error)
}
```

Then the pipeline can mix native and WASM compressors:

```text
detect content type
  │
  ├── native JSON compressor
  ├── native log compressor
  ├── native code compressor
  └── WASM plugin compressor
        │
        ▼
validate token savings
        │
        ▼
apply or fallback
```

The initial backend connection keeps the existing native compressor contract intact. A pipeline can
now receive an ordered list of compressors, with native Go compressors first and a WASM compressor
adapter after them. Each proposal still runs through the same UTF-8 and token-savings validation
before Cachy applies it. If a native proposal does not save tokens, the pipeline may try the next
backend; if a WASM plugin returns `keep`, `annotate`, or `reject`, the adapter returns the original
block text so the existing validation path falls back without mutating provider-visible content.

`plugin.WASMCompressor` loads enabled plugin manifests from the lifecycle directory and only runs
plugins whose `compress` capability matches the selected live-zone block content type. The adapter
sends the same selected-block envelope as `cachy plugin test`; it does not send the full request,
provider secrets, filesystem access, or network access.

## WASM Runtime Candidates

Prefer `wazero` for Cachy:

- Pure Go.
- No CGO.
- Easier Windows distribution.
- Good fit for a single-binary product.

Alternatives:

- Wasmtime: powerful and mature, but brings native runtime packaging complexity.
- Wasmer: also capable, but more moving pieces than Cachy needs early.

## Security Model

Default deny.

- No network access from plugins.
- No filesystem access unless explicitly granted.
- No raw provider API keys.
- No full prompt history unless explicitly configured.
- Strict timeout per call.
- Strict memory limit.
- Panic/trap isolation.
- Plugin result is treated as untrusted input.

This is the main reason to use WASM: safe extension without giving arbitrary code full host access.

## Configuration

Example:

```toml
[plugins]
enabled = true
directory = "~/.cachy/plugins"

[[plugins.modules]]
name = "stacktrace-compressor"
path = "~/.cachy/plugins/stacktrace-compressor/plugin.wasm"
enabled = true
content_types = ["log"]
```

The Electron app can manage this directory later, but the Go CLI should work without Electron.

## CLI Commands

```text
cachy plugin list
cachy plugin inspect stacktrace-compressor
cachy plugin enable stacktrace-compressor
cachy plugin disable stacktrace-compressor
cachy plugin test stacktrace-compressor --fixture ./sample.log
```

The initial lifecycle CLI reads plugin directories from `--dir`, then `CACHY_PLUGIN_DIR`, then the
platform state directory under `plugins/`. Each plugin directory contains `manifest.toml`,
`plugin.wasm`, and an `enabled` marker file when active. Enable and disable commands only create or
remove the marker. `plugin test` requires an enabled plugin, sends the fixture text as the selected
block, and runs through the default-deny host without granting extra permissions.

## When To Build It

Build WASM after these are stable:

1. Transparent Go proxy.
2. Native Go compression pipeline.
3. CCR store.
4. Token validation and fallback.
5. Admin/status API.

Then add WASM as a second compressor backend. This keeps v1 shippable and gives Cachy a strong
extension story without making the foundation complicated.
