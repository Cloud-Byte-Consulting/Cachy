---
name: cachy-security-privacy
description: >-
  Review Cachy security and privacy boundaries. Use for API key handling, local admin API
  design, prompt/content privacy, plugin sandboxing, provider forwarding, logging, and
  destructive operations.
---

# Cachy Security And Privacy

Use this skill before adding features that handle prompts, credentials, local files, plugin execution, or provider traffic.

## Principles

- Prompt content is sensitive by default.
- Provider API keys must never be exposed to Electron renderer code or logs.
- Local admin APIs bind to localhost and require an admin token.
- Plugins are untrusted even when installed locally.
- Logs should be useful without leaking secrets or full prompt payloads by default.
- Writes to user configuration should be explicit, reversible, and auditable.

## Review Checklist

- What secrets does this touch?
- What prompt or tool-output content does this store?
- Where is data persisted, and how is retention controlled?
- Can a malicious local webpage or plugin call the admin API?
- Are provider headers filtered correctly?
- Are compressed blocks recoverable only through intended CCR paths?
- Does this feature change trust boundaries for Electron, MCP, or WASM?

## Required For

- admin API changes
- plugin host changes
- storage schema changes
- credential handling
- logging/telemetry changes
- install/uninstall commands
