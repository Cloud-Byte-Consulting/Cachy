# Pod Bundle Analysis For Cachy

Source analyzed:

```text
C:/Users/BittahCriminal/Downloads/pod-bundle-payload
```

The payload is not a ready-to-install Codex plugin. It is a markdown-driven pod bundle toolkit with
Claude-style skills, routing contracts, templates, and PowerShell packaging scripts.

## What Is In The Bundle

```text
.claude/skills/          shared behavior and domain skills
router-skills/trapi/     TRAPI-specific domain skills
router-skills/cordillera Cordillera/Kubernetes/Volcano domain skills
contracts/               skill routing registry
knowledge-base/          architecture notes for selective skill loading and pod zips
templates/pod-bundle/    markdown pod template files
scripts/                 PowerShell scaffold/package scripts
```

There is no `.codex-plugin/plugin.json`, so this should be treated as source material to adapt, not
as an installable Codex plugin.

## Useful Behaviors To Keep

- Markdown-defined pods: behavior is controlled through `pod.md`, `behavior.md`, `sources.md`, and
  `workflows.md` instead of hidden code defaults.
- Selective skill loading: route by domain first, then load only the matching skill.
- Evidence-first investigation: require primary evidence before conclusions.
- Mutation gate: separate read-only analysis from write actions.
- Work loops: sense, route, gate, execute, verify, record.
- Role separation: recon, engineer, QA, judge, scribe.
- Packaging scripts: create one zip per pod with a manifest and SHA256 hashes.
- Progressive disclosure: skill bodies load first, reference files only when needed.

## Not Relevant To Cachy

- TRAPI routing.
- Cordillera routing.
- Volcano scheduling skills.
- Kubernetes domain skills as default product skills.
- .NET/C# domain skills.
- Azure APIM-specific skills.
- Production promotion paths to sibling TRAPI/Cordillera repos.
- `/trapi-router`, `/cordillera-router`, and `/platform-router` naming.

Some Kubernetes/container content may be useful later for Docker deployment docs, but it should not
be part of Cachy’s default project behavior.

## Shared Skills Worth Porting

High value:

- `solutions-architecture`: useful for Cachy architecture decisions and trade-off analysis.
- `llm-agent-orchestration`: directly relevant to agent integrations, context management, caching,
  tool/MCP design, and cost control.
- `building-ai-agents`: useful for Codex/Claude integration behavior and agent-tool boundaries.
- `evidence-grounded-investigation`: useful as a general engineering discipline.
- `mutation-gate`: useful for guarding destructive repo, system, or deployment changes.
- `planning-and-task-breakdown`: useful for large implementation phases.
- `test-driven-development`: useful for Go proxy, TypeScript app, and plugin host behavior.
- `code-review-and-quality`: useful as a pre-merge review discipline.
- `documentation-and-adrs`: useful for preserving architecture decisions.
- `frontend-ui-engineering`: useful for the Electron companion app.
- `clean-code-typescript`: useful for Electron and optional TypeScript SDK.
- `powershell-scripting`: useful for Windows packaging and installer support.

Medium value:

- `technical-program-management`: useful for roadmap/release planning.
- `security-operations-mitre-attack`: useful for threat modeling and security operations, but it
  should be narrowed for Cachy rather than loaded broadly.
- `co-operating-model`: useful as inspiration, but it should be rewritten around Cachy’s workflow.

Drop or replace:

- `trapi-router`
- `cordillera-router`
- `platform-router` as currently written

## Domain Skills To Create For Cachy

Instead of TRAPI/Cordillera, Cachy should define its own routing domains:

```text
cachy-core-runtime
  Go proxy, provider routing, streaming, CCR, compression pipeline.

cachy-desktop-app
  Electron companion app, local admin API, UI state, packaging.

cachy-plugin-system
  WASM plugin host, plugin manifests, sandboxing, compressor contracts.

cachy-agent-integrations
  Codex, Claude, MCP, OpenAI-compatible local servers.

cachy-release-engineering
  Cross-platform builds, Docker, Winget/Scoop/Homebrew, SBOM, signing.

cachy-security-privacy
  API key handling, local admin API, prompt privacy, plugin sandbox policy.
```

## Proposed Cachy Skill Routing Contract

```yaml
version: 1
routers:
  cachy-core:
    owns: Go proxy, compression, CCR, provider compatibility, streaming
    skills:
      - cachy-core-runtime
      - llm-agent-orchestration
      - test-driven-development
      - code-review-and-quality

  cachy-desktop:
    owns: Electron app, local dashboard, user onboarding, admin API UX
    skills:
      - cachy-desktop-app
      - frontend-ui-engineering
      - clean-code-typescript
      - test-driven-development

  cachy-extensions:
    owns: WASM plugins, MCP tools, SDK adapters
    skills:
      - cachy-plugin-system
      - building-ai-agents
      - llm-agent-orchestration
      - cachy-security-privacy

  cachy-ops:
    owns: packaging, release, install, cross-platform support, SBOM
    skills:
      - cachy-release-engineering
      - powershell-scripting
      - evidence-grounded-investigation
      - documentation-and-adrs
```

## Proposed Cachy Pod

Create a `cachy-core-platform` pod:

```text
pods/cachy-core-platform/
  README.md
  pod.md
  behavior.md
  sources.md
  workflows.md
```

The pod should track:

- Go proxy implementation.
- Electron companion app.
- WASM plugin system.
- Provider compatibility.
- Cross-platform packaging.
- Security/privacy decisions.
- ADRs and release readiness.

## Customization Strategy

1. Convert the payload into a real Codex plugin only after selecting the useful skills.
2. Do not copy TRAPI/Cordillera router skills into Cachy.
3. Rewrite shared skills where they contain external org assumptions.
4. Keep the pod markdown contract and packaging idea.
5. Add Cachy-specific skills under a new plugin, likely `cachy-engineering`.
6. Keep Cachy repo docs and plugin skills separate:
   - repo docs explain product architecture;
   - plugin skills guide agent behavior while working on Cachy.

## Recommended First Plugin Shape

```text
cachy-engineering/
  .codex-plugin/
    plugin.json
  skills/
    cachy-core-runtime/
      SKILL.md
    cachy-desktop-app/
      SKILL.md
    cachy-plugin-system/
      SKILL.md
    cachy-agent-integrations/
      SKILL.md
    cachy-release-engineering/
      SKILL.md
    cachy-security-privacy/
      SKILL.md
    evidence-grounded-investigation/
      SKILL.md
    mutation-gate/
      SKILL.md
    documentation-and-adrs/
      SKILL.md
    test-driven-development/
      SKILL.md
```

This gives Cachy the useful operating model without carrying over product-specific noise.

## Next Decisions

- Keep the adapted plugin in the repo-local plugin directory under `plugins/cachy-engineering`.
- Should we rewrite the imported shared skills into shorter Cachy-specific versions, or copy them
  with attribution/license review?
- Should pod zip packaging remain part of Cachy, or should it become a separate internal workflow
  tool?
