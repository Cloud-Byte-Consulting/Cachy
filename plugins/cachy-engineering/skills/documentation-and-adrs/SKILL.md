---
name: documentation-and-adrs
description: >-
  Record Cachy decisions and technical documentation. Use when making architecture choices,
  changing public behavior, defining plugin contracts, choosing dependencies, or documenting
  platform support.
---

# Documentation And ADRs

Use this skill when a decision should survive the current chat.

## ADR Triggers

- Go vs another runtime.
- Electron responsibilities.
- WASM plugin contract.
- provider compatibility decisions.
- security or storage boundaries.
- dependency choices.
- release and packaging strategy.

## ADR Shape

```text
# ADR N: Title

## Status
Proposed | Accepted | Superseded

## Context
What forces and constraints matter?

## Decision
What are we doing?

## Consequences
What improves, what gets harder, and what must be revisited?
```

## Documentation Rules

- Keep docs close to the code or architecture they describe.
- Separate goals, non-goals, and constraints.
- Explain trade-offs, not just the selected path.
- Update docs in the same change that alters behavior.
