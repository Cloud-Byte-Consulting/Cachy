# ADR 0001: Clean-Room Reference Boundary

## Status
Accepted

## Context
Cachy is a clean-room replacement plan for an LLM context optimization proxy. The Headroom
repository may be useful for behavioral reference, but copying source, docs, fixtures, metadata, or
branding would change the legal and attribution posture of this project.

## Decision
Cachy will treat Headroom, including
`http://truenas-scale-1.tail5a208d.ts.net:30008/Cloud-Byte-Consulting/headroom.git`, as
reference-only material. Cachy implementation work follows the decisions recorded in Cachy docs and
ADRs. Any product, architecture, security, licensing, or packaging decision not already defined in
the Cachy repo must be escalated to the project owner before implementation.

## Consequences
Cachy can choose its own implementation, docs, names, package metadata, and assets. This also means
contributors must summarize observed behavior rather than copy source material, and must preserve an
audit trail for decisions that could affect licensing or attribution.
