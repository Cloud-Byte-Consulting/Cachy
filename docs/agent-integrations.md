# Agent Integrations

Cachy integration commands configure supported agent clients to route provider traffic through the
local Cachy proxy. Integration setup must be reversible, diagnosable, and limited to Cachy-owned
changes.

## Codex

Codex integration manages a bounded block in Codex `config.toml`:

```text
# BEGIN CACHY CODEX INTEGRATION
...
# END CACHY CODEX INTEGRATION
```

Supported commands:

```text
cachy integrations codex install --proxy-base-url http://127.0.0.1:8787/v1
cachy integrations codex repair --proxy-base-url http://127.0.0.1:8787/v1
cachy integrations codex uninstall
```

Use `--dry-run` to print the intended full config content without writing:

```text
cachy integrations codex install --dry-run
```

Use `--config <path>` to target a specific Codex config file. When omitted, Cachy resolves the
config path from `CODEX_HOME/config.toml`, or from `$HOME/.codex/config.toml` when `CODEX_HOME` is
not set.

Install and repair add or replace only the managed Cachy block. Uninstall removes only that block and
preserves unrelated Codex configuration. If an existing top-level `model_provider` is present outside
the managed block, Cachy preserves it and adds only the `model_providers.cachy` provider block so it
does not silently override a user-selected provider.

The managed provider points Codex at Cachy's OpenAI-compatible proxy endpoint and uses
`OPENAI_API_KEY` as the provider credential environment variable. Cachy does not write provider API
keys into Codex config.

## Claude Code

Claude Code integration manages Cachy-owned settings in Claude `settings.json`. It sets
`env.ANTHROPIC_BASE_URL` only when that value is absent or was previously managed by Cachy.

Supported commands:

```text
cachy integrations claude install --anthropic-base-url http://127.0.0.1:8787
cachy integrations claude repair --anthropic-base-url http://127.0.0.1:8787
cachy integrations claude uninstall
```

Use `--dry-run` to print the intended full settings JSON without writing:

```text
cachy integrations claude install --dry-run
```

Use `--settings <path>` to target a specific Claude settings file. When omitted, Cachy resolves the
settings path from `CLAUDE_HOME/settings.json`, or from `$HOME/.claude/settings.json` when
`CLAUDE_HOME` is not set.

Install and repair preserve unrelated Claude settings. Uninstall removes Cachy metadata and removes
`env.ANTHROPIC_BASE_URL` only when it still matches the Cachy-managed value. If an existing
`ANTHROPIC_BASE_URL` is already configured by the user, Cachy preserves it and records only Cachy
metadata so it does not silently redirect Claude traffic.

## Endpoint Recipes

Endpoint recipes generate setup guidance for local OpenAI-compatible backends. The generated output
keeps the same shape for every backend: clients point at Cachy, and Cachy points at the selected
target.

Supported recipes:

```text
ollama
llama.cpp
lm-studio
vllm
localai
custom
```

Examples:

```text
cachy integrations recipe ollama
cachy integrations recipe lm-studio --cachy-base-url http://127.0.0.1:8787
cachy integrations recipe custom --target http://10.0.0.5:9000
```

Recipe output includes:

```text
cachy proxy --listen 127.0.0.1:8787 --target <backend-url>
OPENAI_BASE_URL=http://127.0.0.1:8787/v1
CACHY_TARGET_BASE_URL=<backend-url>
```

## MCP Registration

MCP registration output is a JSON snippet that starts Cachy as a local proxy process. It does not
install unrelated MCP servers or companion services.

```text
cachy integrations mcp --target http://127.0.0.1:11434 --listen 127.0.0.1:8787
```

The generated snippet uses:

```json
{
  "mcpServers": {
    "cachy": {
      "command": "cachy",
      "args": ["proxy", "--listen", "127.0.0.1:8787", "--target", "<backend-url>"]
    }
  }
}
```

## Doctor Diagnostics

`cachy doctor` reports the local proxy target, listen address, provider credential presence, and
integration state for Codex, Claude Code, and MCP registration generation.

```text
cachy doctor --target http://127.0.0.1:11434 --listen 127.0.0.1:8787
```

The integration checks use the same default paths as the installer commands:

- Codex: `CODEX_HOME/config.toml`, or `$HOME/.codex/config.toml`.
- Claude Code: `CLAUDE_HOME/settings.json`, or `$HOME/.claude/settings.json`.

When a managed integration is missing or incomplete, doctor output includes the matching install or
repair command with `--dry-run` so the user can inspect the planned change before writing client
configuration. Doctor output may name environment variable keys, paths, and generated commands, but
must not print provider credential values or URL credentials.

## Out Of Scope

- Installing unrelated semantic-code services.
- OS auto-start.
- Writing provider credentials into client config.
