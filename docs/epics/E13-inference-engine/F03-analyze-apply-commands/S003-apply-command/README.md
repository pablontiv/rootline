---
estado: Specified
tipo: historia
---
# S003: Apply Command

**Feature**: [F03 Analyze & Apply Commands](../README.md)
**Capacidad**: `rootline apply` ejecuta inferencias aprobadas del report
**Cubre**: Postcondicion P2 del Epic

## Antes / Despues

**Antes**: Las inferencias del analyze son informativas — no hay forma de aplicarlas automaticamente. `rootline fix` corrige errores de validacion, pero no aplica inferencias nuevas.

**Despues**: `rootline apply report.json` lee el report de analyze y aplica inferencias aprobadas: extiende enums en .stem, añade campos required, migra valores. Similar a `rootline fix` pero opera sobre inferencias, no errores de validacion.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline apply report.json` lee report y aplica inferencias
- [ ] Inferencias de tipo `extend_enum` añaden valor al enum en .stem
- [ ] Inferencias de tipo `add_required_field` añaden campo a required en .stem
- [ ] Inferencias con `requires_agent: true` se saltan con warning
- [ ] `go test ./... -race` pasa verde

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV2: Contratos JSON mantienen `"version": 1`
  - Verificar: .stem modificados siguen siendo v2 validos
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-apply-schema-mods.md) | Implementar aplicacion de modificaciones de schema |
| [T002](T002-apply-data-corrections.md) | Implementar aplicacion de correcciones de datos |
| [T003](T003-apply-integration-tests.md) | Tests de integracion para apply |

## Fuente de verdad

- `internal/fix/fix.go` — referencia para rewrite de frontmatter
- `internal/proposal/proposal.go` — Proposal struct, tipos de propuestas
- `cmd/rootline/fix.go` — referencia para comando similar
