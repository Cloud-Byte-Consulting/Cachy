# ADR 0005: WASM Plugin Deferral

## Status
Accepted

## Context
WASM plugins are useful for custom compressors, redaction, classification, and policy modules, but
they add a sandboxing and validation boundary. Building them before the native proxy, compression,
CCR, and admin contracts are stable would make the foundation harder to test.

## Decision
Cachy will defer WASM plugins until after the transparent proxy, native compression pipeline, CCR
store, token validation, and admin/status API are stable. The default plugin host will use a
default-deny model: no network, filesystem, secrets, or full prompt history unless a future decision
explicitly grants a capability. `wazero` is the preferred runtime unless a stronger reason is
documented.

## Consequences
The v1 product can ship with native Go compressors first. Plugin output is treated as untrusted
input and must pass Go validation before application. Plugin CLI commands, manifests, permissions,
timeouts, memory limits, and malicious fixture tests are required before plugins become an
extension lane.
