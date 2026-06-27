---
name: cachy-plugin-system
description: >-
  Design and implement Cachy's WASM/plugin extension system. Use for plugin manifests,
  sandboxing, compressor contracts, policy modules, plugin lifecycle commands, and host
  validation.
---

# Cachy Plugin System

WASM is an optional sandboxed extension lane. Go remains the host and decision-maker.

## Principles

- Default deny: no network, filesystem, secrets, or full prompt history unless explicitly granted.
- Plugins receive selected live-zone blocks, not whole requests by default.
- Plugin output is untrusted input; Go validates every result.
- Built-in Go compressors come first; WASM is for custom compressors, redaction, classification, and enterprise policy.
- Use `wazero` unless a stronger reason appears; pure Go keeps Windows packaging clean.

## Contract

Plugins may propose:

- `keep`
- `replace`
- `annotate`
- `reject`

Go must validate:

- UTF-8 correctness
- output size limits
- timeout and memory limits
- token savings where required
- protected fields untouched
- cache hot zone untouched
- CCR marker policy controlled by Go

## Workflow

1. Define the plugin manifest and host capability boundary first.
2. Add host tests with malicious, slow, oversized, and invalid-output plugins.
3. Keep the native compressor interface and WASM compressor interface aligned.
4. Expose CLI commands for list, inspect, enable, disable, and test.
