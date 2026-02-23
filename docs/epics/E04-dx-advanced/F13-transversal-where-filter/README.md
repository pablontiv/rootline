---
tipo: feature
---
# F13: Transversal --where Filter

**Epic**: [E04](../README.md)
**Objetivo**: Todos los comandos transversales (tree, stats, validate, graph) soportan `--where` para filtrado declarativo de records, igual que query
**Beneficio**: Elimina workarounds multi-comando para filtrar vistas (ej: `/roadmap pending` pasa de 5 pasos a 1 comando)
**Milestone**: `rootline tree docs/epics/ --where "estado != 'Completed'" -o table` retorna arbol filtrado

## Scope

**In**: Shared filter helper, --where flag en tree/stats/validate/graph, refactor de query para usar el helper, documentacion y skills alignment
**Out**: Nuevos operadores de query, filtrado por body content, filtrado en `describe`

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-core-implementation/) | Core Implementation | Shared filter helper + --where en los 5 comandos transversales |
| [S002](S002-docs-skills-alignment/) | Docs & Skills Alignment | Documentacion CLI, CLAUDE.md, y skills actualizados con --where |

## Dependencias

- F12 Estado Standardization completado (valores de estado en ingles)

## Fuente de verdad

- `cmd/rootline/` (archivos de comandos)
- `internal/query/expr_eval.go` (CompileWhere, MatchRecord)
- `README.md`, `CLAUDE.md`, `docs/query.md`, `docs/graph.md`
