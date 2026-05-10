---
estado: Completed
tipo: task
---
# T003: Remove Claude marketplace publishing

**Outcome**: [Decommission Pi, MCP, and marketplace publishing](README.md)

## Preserva

- .claude/skills/** remains available for local agent guidance.
- Local development hooks unrelated to marketplace publication remain intact unless they only exist for marketplace sync.

## Contexto

The user clarified that only publication should be removed, not the local .claude skill assets. Prior scouting identified .github/workflows/publish-marketplace.yml and docs/marketplace-pipeline.md as the active publishing surface.

## Alcance

**In**:
1. Delete or disable .github/workflows/publish-marketplace.yml.
2. Remove active marketplace pipeline docs and README index references.
3. Remove hook steps whose only purpose is marketplace publication/sync, if present.

**Out**:
1. Do not delete .claude/skills/**.
2. Do not remove local Claude skill usage documentation unless it specifically describes marketplace publishing.
3. Do not modify external marketplace repositories.

## Estado inicial esperado

Rootline has a publish-marketplace GitHub workflow, docs/marketplace-pipeline.md, local .claude skills, and hooks that mention skill syncing.

## Criterios de Aceptación

- test ! -e .github/workflows/publish-marketplace.yml returns exit 0, or the workflow is demonstrably removed from active CI triggers.
- rg 'agent-marketplace|publish-marketplace|MARKETPLACE_TOKEN|marketplace-pipeline' returns no active publishing workflow/docs references.
- test -d .claude/skills returns exit 0.

## Fuente de verdad

- .github/workflows/publish-marketplace.yml
- docs/marketplace-pipeline.md
- .githooks/pre-push
- .githooks/post-merge
- .claude/skills/
