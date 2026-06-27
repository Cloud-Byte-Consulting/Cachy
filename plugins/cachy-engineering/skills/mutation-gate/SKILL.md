---
name: mutation-gate
description: >-
  Gate risky Cachy mutations. Use before destructive filesystem actions, repo history
  changes, credential/config writes, package publishing, release changes, or
  deployment-affecting operations.
---

# Mutation Gate

Use this skill before risky writes.

## Gate Levels

- Allow: low-risk, reversible, scoped, and requested.
- Revise: action is reasonable but needs a safer scope or backup.
- Escalate: action affects credentials, releases, deployments, public packages, or user/global config.
- Block: action is destructive, unrelated, legally unsafe, or lacks necessary context.

## Always Escalate Before

- deleting user data
- changing git history
- publishing packages or containers
- modifying global Codex/Claude config
- rotating or storing secrets
- changing license files
- enabling auto-start services

## Workflow

1. Identify what will change.
2. Identify blast radius and rollback path.
3. Prefer dry runs, backups, and scoped writes.
4. Proceed only when the risk level matches the user's authorization.
