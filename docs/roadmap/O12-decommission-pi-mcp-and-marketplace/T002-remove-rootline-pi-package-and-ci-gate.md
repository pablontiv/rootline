---
estado: Specified
tipo: task
---
# T002: Remove Rootline-owned Pi package and CI gate

**Outcome**: [Decommission Pi, MCP, and marketplace publishing](README.md)

## Preserva

- Rootline remains the CLI provider consumed by external Pi tooling.
- No compatibility stub or duplicate Pi package remains in Rootline.

## Contexto

The user chose a clean migration: Pinata owns the Pi integration, with no compatibility shim in Rootline. Current Rootline package lives under integrations/pi and CI has a pi-extensions job/release gate.

## Alcance

**In**:
1. Delete integrations/pi/** from Rootline.
2. Remove Rootline CI jobs, release gates, smoke tests, and docs that assume integrations/pi is canonical.
3. Update active Rootline docs to say Pi integration now belongs outside Rootline if any active mention remains necessary.

**Out**:
1. Do not leave a stub package or deprecation README under integrations/pi.
2. Do not publish or install Pinata from Rootline CI.
3. Do not remove Rootline CLI commands used by the Pi integration unless covered by another approved task.

## Estado inicial esperado

Rootline contains integrations/pi/package.json, extensions, prompts, tests, smoke tests, README release strategy, and a pi-extensions CI job.

## Criterios de Aceptación

- test ! -e integrations/pi returns exit 0 in the Rootline repo.
- Rootline CI workflow no longer references integrations/pi or pi-extensions.
- rg 'integrations/pi|pi-rootline|pi-extensions' returns no active Rootline packaging references outside historical roadmap records intentionally retained.
- just check or equivalent Rootline checks pass.

## Fuente de verdad

- integrations/pi/
- .github/workflows/ci.yml
- README.md
- docs/roadmap/O02-design-pi-extension-architecture/
- docs/roadmap/O08-productionize-testing-release-and-adoption/
