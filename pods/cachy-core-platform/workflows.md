# Workflows

## Runtime Feature

### Trigger

A change to Go proxy, provider routing, streaming, compression, CCR, or token accounting.

### Steps

1. Use `cachy-core-runtime`.
2. Gather evidence from current code and protocol docs.
3. Add or update tests around provider-visible behavior.
4. Implement the smallest runtime change.
5. Run focused Go tests, then `go test ./...`.
6. Record architecture implications if behavior changes.

### Exit Criteria

- Tests pass.
- Provider-visible behavior is documented.
- Rollback path is clear.

## Desktop Feature

### Trigger

A change to Electron UI, local setup, dashboard, provider config, or binary supervision.

### Steps

1. Use `cachy-desktop-app`.
2. Confirm whether the capability belongs in Go or Electron.
3. Add admin API or CLI support first if needed.
4. Implement UI with loading, empty, error, and disabled states.
5. Test on platform-specific path assumptions.

### Exit Criteria

- UI does not own secrets or proxy traffic.
- Go binary remains usable without Electron.

## Extension Feature

### Trigger

A change to WASM, MCP, plugins, custom compressors, or plugin security.

### Steps

1. Use `cachy-plugin-system` and `cachy-security-privacy`.
2. Define capability boundary and manifest changes.
3. Add malicious/invalid plugin tests.
4. Validate plugin output in Go before applying it.
5. Document any new permission.

### Exit Criteria

- Default deny preserved.
- Timeouts, memory limits, and validation paths are tested.

## Release Readiness

### Trigger

Preparing a binary, Docker image, Electron package, or public release.

### Steps

1. Use `cachy-release-engineering`.
2. Build all target platforms.
3. Generate checksums and SBOMs.
4. Run smoke tests for install and health checks.
5. Record release notes and known limitations.

### Exit Criteria

- Artifacts are reproducible enough for review.
- Checksums and SBOMs exist.
- Platform caveats are documented.
