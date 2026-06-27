# ADR 0004: Electron Companion Role

## Status
Accepted

## Context
Cachy may include a desktop app to help users start, configure, diagnose, and understand the local
proxy. The desktop app will interact with provider settings, logs, integration setup, and local
status, so trust boundaries need to be explicit before implementation.

## Decision
Electron is a companion controller, not the trusted request path. The Go binary owns proxying,
compression, provider forwarding, persistence, secrets, and admin/config validation. Electron may
discover or supervise the Go binary and communicate through stable CLI or localhost admin APIs.
Provider API keys and admin tokens must not be exposed directly to renderer code.

## Consequences
The desktop app remains optional, and Cachy stays useful for headless and server workflows. UI work
must wait for stable CLI/admin surfaces for security-sensitive behavior. Electron tests must cover
loading, empty, error, disabled, and permission-denied states, plus secret-boundary behavior.
