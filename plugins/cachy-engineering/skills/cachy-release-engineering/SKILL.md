---
name: cachy-release-engineering
description: >-
  Build Cachy's cross-platform release system. Use for Windows/macOS/Linux builds, Docker
  images, Electron packaging, SBOMs, checksums, signing, installers, and release validation.
---

# Cachy Release Engineering

Use this skill when changing packaging, CI, release workflows, or install behavior.

## Platform Promise

Users should be able to download one archive, put `cachy` on PATH, and run it without Python, Rust, Node, or a compiler.

## Required Targets

- `windows/amd64`
- `windows/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`

## Workflow

1. Keep default builds CGO-free unless a deliberate exception is documented.
2. Cross-compile all CLI targets in CI.
3. Produce checksums for release artifacts.
4. Generate SBOMs for source, binary, Docker, and Electron distributions where practical.
5. Test installer behavior on Windows, macOS, and Linux.
6. Keep Docker multi-arch images for `linux/amd64` and `linux/arm64`.

## Packaging Channels

- ZIP for Windows.
- tar.gz for macOS/Linux.
- Docker image for server/headless use.
- Winget and Scoop later.
- Homebrew later.
- deb/rpm later.
