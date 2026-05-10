---
estado: Completed
tipo: task
---
# T004: Clean active Rootline docs and agent contracts

**Outcome**: [Decommission Pi, MCP, and marketplace publishing](README.md)

## Preserva

- Historical roadmap records can remain historical unless they break validation or create active instructions.
- Rootline roadmap validation remains green.

## Contexto

The user selected documentation policy C: remove active MCP docs and active references, but avoid rewriting historical roadmap records unless needed. This task consolidates the final documentation pass after code-surface removals.

## Alcance

**In**:
1. Update README.md command lists, status claims, docs index, and installation guidance.
2. Update CLAUDE.md and .claude/skills/rootline references that instruct agents to use removed MCP files or marketplace publishing.
3. Delete active docs/json-rpc.md if it only documents removed MCP support, or otherwise ensure it is not linked as an active product doc.
4. Run roadmap/document validation and fix stale active references.

**Out**:
1. Do not rewrite completed/historical roadmap records solely to erase history.
2. Do not remove .claude/skills/** local assets.
3. Do not document MCP as deprecated runtime support; it is removed.

## Estado inicial esperado

README.md, docs/json-rpc.md, CLAUDE.md, .claude skills, and roadmap records contain active and historical references to MCP, Pi co-location, and marketplace publishing.

## Criterios de Aceptación

- README.md no longer advertises MCP, rootline serve, Claude Desktop MCP setup, or Rootline-owned Pi package installation.
- CLAUDE.md and active .claude skills no longer point agents at internal/mcp/tools.go or removed marketplace publishing workflows.
- rootline validate --all docs/roadmap/ returns exit 0.
- roadmapctl check --repo /home/shared/rootline --roadmap-root docs/roadmap --output json --strict returns status ok.

## Fuente de verdad

- README.md
- CLAUDE.md
- .claude/skills/rootline/
- docs/json-rpc.md
- docs/roadmap/
- docs/marketplace-pipeline.md
