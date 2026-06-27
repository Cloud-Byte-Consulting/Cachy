# Agent Integration MVP Readiness

## Status

Ready for the current MVP scope.

## Scope

The agent integration MVP covers reversible setup and diagnostics for:

- Codex configuration through a Cachy-managed `config.toml` block.
- Claude Code configuration through Cachy-managed `settings.json` metadata.
- MCP registration snippets that launch Cachy as a local proxy process.
- OpenAI-compatible endpoint recipes for Ollama, llama.cpp server, LM Studio, vLLM, LocalAI, and
  custom targets.
- `cachy doctor` checks for integration state, local proxy listen address, provider target,
  provider reachability, and credential presence.

## Validation Evidence

- Issue #29 closed after Codex install, repair, uninstall, and dry-run command support landed.
- Issue #30 closed after Claude Code install, repair, uninstall, and dry-run command support landed.
- Issue #31 closed after MCP registration and endpoint recipe output landed.
- Issue #32 closed after integration-specific doctor diagnostics and dry-run repair guidance landed.
- Issue #4 closed for the CLI and platform foundation required by the integration commands.
- Issue #10 closed for the admin and observability baseline used by local diagnostics.

Before closing this readiness item, run:

```text
go test ./...
go vet ./...
git diff --check
```

## Supported Behavior

Codex setup writes only a bounded Cachy-owned block, preserves unrelated config, and does not write
provider credentials. If a user already has a top-level Codex `model_provider`, Cachy preserves that
selection and adds the Cachy provider definition without silently changing the active provider.

Claude Code setup preserves unrelated settings, writes `env.ANTHROPIC_BASE_URL` only when absent or
previously managed by Cachy, and removes only Cachy-managed settings during uninstall.

Endpoint recipes and MCP registration are generated output. They do not install unrelated services,
semantic-code servers, OS startup entries, or external MCP companion tools.

Doctor diagnostics report the state of supported integrations and print dry-run install or repair
commands for missing or incomplete setup. Diagnostics may name paths, environment variable keys, and
commands, but must not print provider credential values or URL credentials.

## Known Boundaries

- OS auto-start is out of scope.
- Installing unrelated semantic-code services is out of scope.
- Writing provider credentials into client configuration is out of scope.
- Electron companion setup screens are part of the later desktop companion wave.
- TypeScript SDK adapters remain part of the later SDK and local LLM polish phase.
