---
name: test-driven-development
description: >-
  Drive Cachy changes with tests. Use when implementing or changing Go runtime behavior,
  Electron UI logic, WASM host behavior, integrations, packaging scripts, or bug fixes.
---

# Test-Driven Development

Use this skill for implementation and bug fixes.

## Principles

- Characterize current behavior before changing risky code.
- Test provider-visible behavior with fixtures and mock upstreams.
- Keep core compression logic deterministic and easy to unit test.
- Add integration tests for protocol, streaming, and cross-platform paths.
- Test the failure path, not only the happy path.

## Workflow

1. Write or identify the failing test.
2. Make the smallest implementation change.
3. Run focused tests.
4. Refactor only after tests pass.
5. Run broader tests before finishing.

## Test Areas

- Go unit tests for compressors and token validation.
- Go integration tests for provider pass-through and SSE.
- WASM host tests for malicious/invalid plugins.
- Electron UI tests for state and admin API failures.
- Packaging tests for path handling and binary discovery.
