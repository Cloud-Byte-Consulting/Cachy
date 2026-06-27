---
name: cachy-core-runtime
description: >-
  Build Cachy's Go runtime: local proxy, provider routing, streaming, token accounting,
  cache-safe compression, CCR storage, and admin APIs. Use when editing Go proxy code,
  designing provider compatibility, handling OpenAI/Anthropic/local LLM traffic, or
  making runtime architecture decisions.
---

# Cachy Core Runtime

Use this skill for Cachy's native Go binary.

## Principles

- Keep the Go binary useful without Electron.
- Prefer pure-Go dependencies and avoid compiler toolchains for users.
- Preserve provider cache hot zones by mutating only selected live-zone content.
- Treat compression output as a proposal that must pass validation before application.
- Keep passthrough behavior byte-conscious; do not reserialize stable request regions casually.
- Make every provider adapter testable with local fixtures and mock upstreams.

## Workflow

1. Identify the request family: OpenAI chat, OpenAI responses, Anthropic messages, WebSocket, or local OpenAI-compatible.
2. Confirm whether the change touches passthrough, live-zone selection, compression, CCR, auth/header behavior, or observability.
3. Add or update tests before changing provider-visible behavior.
4. Preserve streaming semantics: events, order, status codes, headers, and cancellation behavior.
5. Validate token savings after compression; fallback to original content when savings are absent or uncertain.
6. Document provider-specific quirks in repo docs or an ADR when they affect future design.

## Default Checks

- `go test ./...`
- streaming fixture tests for provider protocol changes
- large payload passthrough tests for compression changes
- Windows path/config tests for runtime filesystem changes
