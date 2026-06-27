---
name: evidence-grounded-investigation
description: >-
  Ground Cachy plans, bugs, and architecture decisions in primary evidence. Use when
  diagnosing behavior, comparing options, validating claims, or preparing implementation
  plans.
---

# Evidence-Grounded Investigation

Use this skill when a claim needs proof before action.

## Evidence Order

1. Local repo files and tests.
2. Runtime command output.
3. Official provider docs.
4. Standards/specifications.
5. Issue trackers or release notes.
6. Secondary sources only when primary sources are unavailable.

## Workflow

1. State the question being investigated.
2. Gather primary evidence.
3. Separate observed facts from inference.
4. Note uncertainty and stale assumptions.
5. Recommend action with citations to files, commands, docs, or test results.

## Anti-Patterns

- Treating README claims as implementation truth without reading code.
- Assuming provider behavior without checking current docs.
- Making release or security decisions from memory.
- Burying uncertainty.
