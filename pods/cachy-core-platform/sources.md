# Sources

Use primary evidence when making Cachy decisions.

| Source | Query/Scope | Required fields | Notes |
|---|---|---|---|
| Cachy repo | local files and git state | path, relevant lines, test output | primary implementation truth |
| Headroom repo | behavioral reference only | source URL, observed behavior, non-copied summary | never implementation authority |
| Go docs | standard library and runtime behavior | version, package, API | use for runtime decisions |
| Provider docs | OpenAI, Anthropic, OpenRouter, local server docs | endpoint, request/response semantics, streaming behavior | verify current behavior |
| Electron docs | app lifecycle, security, packaging | version, API, platform caveats | use for desktop app choices |
| WASM runtime docs | wazero or selected runtime | version, sandbox limits, host calls | use for plugin host choices |
| Release artifacts | CI output, SBOM, checksums | target, hash, build log | required before release decisions |

## Evidence Policy

1. Separate observed facts from inference.
2. Prefer local tests over assumptions.
3. Mark inaccessible sources explicitly.
4. Do not claim provider compatibility without fixtures or docs.
5. Do not make security claims without identifying the trust boundary.
