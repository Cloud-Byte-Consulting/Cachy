# Behavior Rules

## Prioritization

1. Security/privacy boundaries and credential handling.
2. Cross-platform install and runtime correctness.
3. Transparent proxy compatibility.
4. Cache-safe compression and CCR.
5. Electron companion UX.
6. WASM plugin extensibility.
7. Nice-to-have integrations and polish.

## Triage Rules

| Condition | Classification | Action |
|---|---|---|
| Go proxy, streaming, CCR, compression, provider forwarding | cachy-core | Use `cachy-core-runtime` |
| Electron, dashboard, setup, local admin UX | cachy-desktop | Use `cachy-desktop-app` |
| WASM, plugin manifest, sandbox, extension lifecycle | cachy-extensions | Use `cachy-plugin-system` and `cachy-security-privacy` |
| Codex, Claude, MCP, local LLM setup | cachy-extensions | Use `cachy-agent-integrations` |
| CI, binaries, Docker, SBOM, signing, installers | cachy-ops | Use `cachy-release-engineering` |
| Credentials, prompt privacy, logs, admin API, plugin permissions | cachy-security | Use `cachy-security-privacy` |
| Major architecture decision | architecture | Use `documentation-and-adrs` |

## Escalation Rules

Stop and require explicit confirmation before:

1. Publishing packages, images, or releases.
2. Changing license files or attribution.
3. Writing global Codex, Claude, shell, or OS auto-start config.
4. Deleting user data, CCR stores, or repo history.
5. Storing or printing provider API keys.
6. Enabling plugin permissions beyond default deny.
7. Making product, architecture, security, licensing, or packaging decisions that are not already
   defined in Cachy docs or ADRs.

## Reference-Only Sources

Headroom may be used as behavioral reference material only. Do not copy its source, docs, fixtures,
metadata, install behavior, or branding into Cachy unless the owner explicitly approves a derivative
path with license and attribution handling.
