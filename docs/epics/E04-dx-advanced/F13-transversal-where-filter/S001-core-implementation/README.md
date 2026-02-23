---
tipo: historia
---
# S001: Core Implementation

**Feature**: [F13 Transversal --where Filter](../README.md)
**Capacidad**: Los comandos tree, stats, validate --all, y graph aceptan --where para filtrar records antes de procesar

## Antes / Despues

**Antes**: Solo `rootline query` tiene `--where`. Para filtrar en tree/stats/graph hay que hacer query primero, extraer paths, y luego ejecutar cada comando por separado. `/roadmap pending` requiere 5 pasos con 3 comandos distintos.

**Despues**: `rootline tree docs/ --where "estado != 'Completed'" -o table` filtra directamente. Todos los comandos transversales comparten la misma infraestructura de filtrado via un helper centralizado. Query refactorizado para usar el mismo helper.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline tree --where` filtra records antes de construir el arbol
- [ ] `rootline stats --where` cuenta solo records que matchean el filtro
- [ ] `rootline validate --all --where` valida solo records filtrados
- [ ] `rootline graph --where` construye grafo solo con records filtrados
- [ ] `rootline query --where` sigue funcionando igual (refactor interno, misma API)
- [ ] Expresiones invalidas retornan error claro en todos los comandos

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-shared-filter-helper.md) | Crear filterRecords() helper compartido en cmd/rootline/filter.go |
| [T002](T002-tree-where-flag.md) | Agregar --where a tree command + fix bug Completado |
| [T003](T003-stats-where-flag.md) | Agregar --where a stats command |
| [T004](T004-validate-where-flag.md) | Agregar --where a validate --all |
| [T005](T005-graph-where-flag.md) | Agregar --where a graph command |

## Fuente de verdad

- `cmd/rootline/` (tree.go, stats.go, validate.go, graph.go, query.go)
- `internal/query/expr_eval.go` (CompileWhere, BuildEnv, MatchRecord)
