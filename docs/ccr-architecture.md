# CCR Architecture

CCR stores original live-zone content locally so a compressed request can carry a small retrieval
marker instead of the full original block. Original content is sensitive by default and remains under
Go runtime control; plugins and logs must not receive full originals unless a later explicit
decision grants that access.

Current status: the local CCR storage foundation is ready for future opt-in proxy integration. The
implemented foundation includes v1 marker parsing/rendering, SHA-256 content addresses,
content-addressed local storage, retention cleanup, size diagnostics, recovery tests, and privacy
tests. Marker injection into provider requests is intentionally not enabled in the proxy path yet.

## Marker Contract

CCR markers use this v1 provider-visible format:

```text
[[cachy-ccr:v1 sha256:<64 lowercase hex characters> bytes:<original byte count>]]
```

Marker properties:

- `sha256` is the content-addressing hash for v1.
- `bytes` records the original content byte length.
- Markers are bounded to 128 bytes.
- Markers contain no prompt text, tool output, provider credentials, file paths, or local storage
  paths.
- Go renders, parses, and validates markers. Plugins may propose compression output, but marker
  creation and CCR lookup remain host-controlled.

## Compatibility Rules

Cachy may emit a marker only when all of these are true:

- The block is selected for compression.
- The block is live, not stable/protected cache-prefix content.
- The provider is currently supported for CCR markers: OpenAI-compatible chat or Anthropic
  messages.
- The original has already been written to the local CCR store.

Stable system, developer, ordinary user, and assistant message content remains protected and must
not be replaced by a marker under the current v1 rules.

## Local Store

The local store uses the platform-aware Cachy state directory:

```text
<state-dir>/ccr/objects/sha256/<first-two-hash-chars>/<sha256-hex>
```

Store behavior:

- Writes are content-addressed and idempotent. Writing the same original content returns the same
  address without rewriting an existing object.
- Object directories are created with owner-only permissions where the platform supports them.
- Object files are written with owner read/write permissions where the platform supports them.
- Reads require a validated CCR address and verify the file still hashes to that address before
  returning content.
- Missing content and corrupt content are reported as distinct errors so callers can decide whether
  to fall back, repair, or surface diagnostics.

## Retention And Diagnostics

CCR retention is local and explicit. Cleanup accepts a policy with optional maximum object age and
maximum total bytes:

- Dry-run cleanup reports what would be removed without deleting objects.
- Confirmed cleanup removes expired objects first, then oldest remaining objects until total stored
  bytes are within policy.
- Cleanup reports removed, failed, kept, and byte totals so admin diagnostics can explain outcomes.
- Delete failures are counted and leave the original object in place.
- Diagnostics report the CCR root path, object count, and total object bytes without reading or
  returning original content.

## Validation Evidence

The current marker tests cover:

- Deterministic SHA-256 addressing and original byte counts.
- Marker render/parse round trips.
- Marker length bounds.
- Rejection of malformed markers, unsupported versions, unsupported hash algorithms, invalid hash
  text, and invalid byte counts.
- Compatibility checks for selected live blocks and unsupported providers.
- Local store writes, reads, duplicate writes, missing content, corrupt content, invalid addresses,
  and platform state-root path construction.
- Empty-store diagnostics, object count and size reporting, cleanup dry-runs, confirmed expired
  cleanup, byte-limit cleanup, and delete failure reporting.
- Recovery by parsed marker address, missing unrelated content, marker privacy, diagnostics privacy,
  cleanup-result privacy, and request logging surfaces that omit original content by default.

## Known Limitations

- Markers are not yet wired into the proxy request path.
- Remote sync and cloud storage are out of scope.
