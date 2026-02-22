---
estado: Specified
tipo: historia
cliente: Platform Owner
---
# S004: Graph Schema-Aware Link Filtering

**Feature**: [F05 Dependency Graph](../README.md)
**Capacidad**: graph --check solo evalua links que el .stem define como estructurales, eliminando falsos positivos

## Antes / Despues

**Antes**: `rootline graph --check` parsea todos los `[[...]]` del body como links y checa si el target existe como nodo, sin consultar el schema de links del .stem. Texto como `[[target]]` o `[[A,B,C,A]]` en prosa se reporta como broken links. `validateLinks()` tampoco los atrapa porque `reference` no tiene regla de target. El .stem define `target: "T*"` con glob impreciso via `filepath.Match`.

**Despues**: El graph command carga .stem y pre-filtra links antes de Build(). Solo links cuyo tipo tiene `target` rule en `schema.Rules` se convierten en edges del grafo. `rootline graph --check docs/epics/` retorna 0 broken links. Target patterns usan regexp (Go stdlib) en vez de filepath.Match, permitiendo patterns precisos como `^T\d{3}-`.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline graph --check docs/epics/` retorna exit 0 (0 broken links, 0 cycles)
- [ ] Links tipo `reference` sin target rule en .stem no generan edges en el grafo
- [ ] Links tipo `blocks` con target rule se validan contra regexp pattern
- [ ] Target pattern `^T\d{3}-` matchea `T001-nombre` pero no `target` ni `A,B,C,A`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-graph-load-stem-filter-links.md) | Graph command carga .stem y pre-filtra links antes de Build() |
| [T002](T002-regexp-target-patterns.md) | Cambiar target patterns de filepath.Match glob a regexp |

## Fuente de verdad

- `cmd/rootline/graph.go` — graph command (no carga .stem)
- `internal/graph/graph.go` — Build(), BrokenLinks()
- `internal/rules/validate.go` — validateLinks() con filepath.Match
- `docs/epics/.stem:45-48` — links schema actual
