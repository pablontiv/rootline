---
tipo: feature
---
# E03: Rootline Core & Distribution

**Estado**: In Progress
**Metrica de exito**: Rootline binario disponible via Homebrew y GitHub Releases, MCP server funcional
**Timeline**: 2026-Q1 — en curso

## Intencion

Completar la distribución de rootline como herramienta CLI independiente: MCP server para integración con editores AI, release pipeline automatizada con goreleaser + svu, y distribución via Homebrew tap.

## Features

| Feature | Descripcion |
|---------|-------------|
| [F05 MCP Distribution](F05-mcp-distribution/README.md) | MCP server y release pipeline |
| [F06 GitHub Action](F06-github-action/README.md) | Validacion de docs en CI con annotations en PRs |
| [F07 Claude Code Plugin](F07-claude-code-plugin/README.md) | Skills /validate, /describe, /new-doc para Claude Code |
| [F08 Proposal Engine Fixes](F08-proposal-engine-fixes/README.md) | Fix prioridad entre detectores en proposal engine |
| [F09 Marketplace Distribution Pipeline](F09-marketplace-distribution-pipeline/README.md) | Skills como plugin distribuible via marketplace |
| [F10 Engine API Layer](F10-engine-api-layer/README.md) | Extraer business logic a API reutilizable |
| [F11 Aggregate Auto-Generation & Command Consolidation](F11-aggregate-command-consolidation/README.md) | Auto-generar aggregate para enums jerárquicos, unificar validate+doctor |
