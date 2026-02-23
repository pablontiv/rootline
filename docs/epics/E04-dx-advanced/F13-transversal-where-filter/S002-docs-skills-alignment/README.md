---
tipo: historia
---
# S002: Docs & Skills Alignment

**Feature**: [F13 Transversal --where Filter](../README.md)
**Capacidad**: Documentacion CLI, CLAUDE.md, y skills reflejan que --where esta disponible en todos los comandos transversales

## Antes / Despues

**Antes**: La documentacion solo menciona `--where` en el contexto de `rootline query`. Skills como `/roadmap pending` usan workarounds multi-comando. CLAUDE.md no menciona que --where existe en tree/stats/graph/validate.

**Despues**: README.md, docs/query.md, docs/graph.md, CLAUDE.md, y todos los skills relevantes documentan `--where` como flag universal de comandos transversales. `/roadmap pending` simplificado a 2 pasos.

## Criterios de Aceptacion (semanticos)

- [ ] README.md tabla CLI muestra --where en tree, stats, graph, validate
- [ ] docs/query.md tiene cross-reference a todos los comandos con --where
- [ ] CLAUDE.md seccion Rootline menciona --where en comandos transversales
- [ ] `/roadmap pending` en SKILL.md usa `rootline tree --where` directamente

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-update-cli-docs.md) | Actualizar README.md y docs/ con --where en comandos transversales |
| [T002](T002-update-skills-claude.md) | Actualizar CLAUDE.md y skills con --where |

## Fuente de verdad

- `README.md`, `CLAUDE.md`
- `docs/query.md`, `docs/graph.md`
- `.claude/skills/roadmap/SKILL.md`, `.claude/skills/roadmap/epic-guide.md`
