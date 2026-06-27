---
name: cachy-agent-integrations
description: >-
  Build Cachy integrations for Codex, Claude, MCP, and OpenAI-compatible local model
  servers. Use when implementing wrapper commands, MCP registration, environment setup,
  or compatibility recipes.
---

# Cachy Agent Integrations

Use this skill for client setup and agent-facing behavior.

## Principles

- Install Cachy integrations only; do not install unrelated semantic-code services.
- Prefer config-file, MCP, and environment-based setup over shell-hook assumptions.
- Windows, macOS, and Linux must all have a first-class path.
- Keep local LLM support conservative: optimize context pressure and latency, not token billing claims.
- Make install commands reversible and diagnosable.

## Supported Integration Targets

- Codex
- Claude Code
- MCP tools
- OpenAI-compatible endpoints
- Ollama
- llama.cpp server
- LM Studio
- vLLM
- LocalAI

## Workflow

1. Detect OS and existing client configuration.
2. Show intended changes before mutating user config when practical.
3. Write config through platform-aware paths.
4. Provide `doctor` checks for env vars, ports, target URLs, and credentials.
5. Add uninstall or repair behavior for every install path.
