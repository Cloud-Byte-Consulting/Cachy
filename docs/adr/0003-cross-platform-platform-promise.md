# ADR 0003: Cross-Platform Platform Promise

## Status
Accepted

## Context
Cachy is expected to support Windows, macOS, and Linux from the beginning. Platform-specific path,
packaging, and install behavior can easily become hard to unwind if Unix-only assumptions enter the
core runtime.

## Decision
Cachy will support these default CLI targets:

- `windows/amd64`
- `windows/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`

The default binary must work without Python, Rust, Node, or a compiler. Config, state, cache, and
secret handling must use platform-aware helpers rather than hard-coded Unix paths.

## Consequences
CI and release work must include cross-compilation and platform path tests early. Dependencies that
require native toolchains are disfavored for default builds. Installer and integration work must
have first-class Windows paths rather than relying on shell hooks.
