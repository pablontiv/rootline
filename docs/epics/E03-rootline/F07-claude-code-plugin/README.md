---
tipo: feature
---
# F07: Claude Code Plugin

**Epic**: [E03](../README.md)
**Objetivo**: Developers usando Claude Code pueden validar, inspeccionar y crear documentacion estructurada via skills dedicados
**Beneficio**: AI-assisted authorship — el LLM conoce el schema y puede scaffold docs validos sin trial-and-error
**Milestone**: `/validate docs/epics/` ejecuta validacion y muestra errores formateados; `/new-doc` crea documento con frontmatter correcto

## Scope

**In**: Plugin scaffold (plugin.json), 3 skills (/validate, /describe, /new-doc)
**Out**: Hooks de auto-validacion, agents, MCP integration (eso es F05), marketplace publishing

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Core Skills](S001-core-skills/) | /validate, /describe, /new-doc funcionan como skills de Claude Code |

## Dependencias

- rootline binary en PATH (instalacion manual o goreleaser release)

## Fuente de verdad

- `.claude/skills/roadmap/` (patron existente de skill local)
- rootline CLI commands (validate, describe, new)
