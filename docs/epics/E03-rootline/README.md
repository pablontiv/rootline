---
estado: In Progress
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
