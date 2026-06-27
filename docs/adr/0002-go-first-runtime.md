# ADR 0002: Go-First Runtime

## Status
Accepted

## Context
Cachy needs to be easy to install on Windows, macOS, Linux, servers, dev containers, and headless
machines. The architecture docs identify the runtime as responsible for proxying, provider routing,
streaming, compression orchestration, CCR storage, metrics, and install commands.

## Decision
Cachy will use a Go-first runtime. The Go binary owns the trusted request path and remains useful
without Electron. Runtime dependencies should be pure Go by default, with CGO or native toolchains
used only behind explicit decisions.

## Consequences
Users can run Cachy without Python, Rust, Node, or a compiler. Go must own provider-visible behavior
and test coverage for proxying, streaming, storage, and security-sensitive decisions. Optional SDKs
or desktop surfaces may exist, but they must not replace the Go runtime.
