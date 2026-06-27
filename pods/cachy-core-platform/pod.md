---
id: cachy-core-platform
title: Cachy Core Platform
description: Operating pod for Cachy's runtime, desktop app, extension system, integrations, and release engineering.
owners:
  - alias: Cloud Byte Consulting
    role: owner
scope:
  systems:
    - go-runtime
    - electron-app
    - wasm-plugins
    - codex-claude-integrations
    - cross-platform-release
routing:
  defaultRouter: cachy-core
  allowedRouters:
    - cachy-core
    - cachy-desktop
    - cachy-extensions
    - cachy-ops
writePolicy:
  defaultMode: read-only-first
  requiresMutationGate: true
---

# Pod Definition

## Intent

Guide Cachy work so implementation stays portable, evidence-grounded, testable, and free of
unwanted inherited product assumptions.

## Success Signals

1. The Go binary runs on Windows, macOS, and Linux without requiring Python, Rust, Node, or a compiler.
2. The proxy supports transparent provider pass-through before compression is enabled.
3. Compression preserves cache hot zones and falls back safely when token savings are uncertain.
4. Electron remains a companion controller, not the trusted proxy path.
5. WASM remains an optional plugin sandbox, not a v1 dependency.
6. Decisions that affect architecture, security, or packaging are recorded.

## Out Of Scope

- Installing unrelated semantic-code companion services.
- Carrying over external project names, routers, or product-specific assumptions.
- Adding Kubernetes or cloud operations skills to the default workflow unless the task is explicitly about deployment.
