---
name: cachy-desktop-app
description: >-
  Build Cachy's Electron companion app. Use when designing or implementing the dashboard,
  setup flows, provider configuration, local admin API UX, logs, savings views, or bundled
  Go-binary supervision.
---

# Cachy Desktop App

Electron is a companion app, not the trusted request path.

## Principles

- The Go binary owns proxying, compression, storage, and secrets.
- The desktop app starts, stops, configures, and explains Cachy.
- The app should discover an installed `cachy` binary before using a bundled one.
- Never expose provider API keys directly to renderer code.
- Keep the first screen operational: status, provider, savings, errors, and next action.

## Expected Views

- Dashboard: proxy status, active provider, savings, recent failures.
- Sessions: request history with before/after token counts.
- Providers: OpenAI, Anthropic, OpenRouter, Ollama, LM Studio, vLLM, custom endpoint.
- Integrations: Codex, Claude, MCP, shell environment.
- Storage: CCR size, retention, cleanup.
- Diagnostics: ports, versions, config paths, network checks.

## Workflow

1. Decide whether a feature belongs in Go CLI/admin API or Electron UI.
2. Prefer invoking stable JSON CLI/admin endpoints over direct file edits.
3. Add UI states for loading, empty, error, disabled, and permission-denied.
4. Keep UI dense and operational rather than marketing-oriented.
5. Test on Windows, macOS, and Linux packaging assumptions.
