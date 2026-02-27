---
estado: Completed
tipo: historia
---
# S001: Aggregate Expression Generator

**Feature**: [F11 Aggregate Auto-Generation & Command Consolidation](../README.md)
**Capacidad**: Librería que genera expresiones aggregate a partir de valores enum usando clasificación por keywords multilingüe
**Cubre**: Generador core reutilizable por init, migrate y fix

## Antes / Despues

**Antes**: No existe generador de aggregate. Init y migrate producen .stem sin aggregate para campos enum compartidos entre niveles jerárquicos. El usuario debe escribir manualmente las expresiones aggregate.

**Despues**: `migrate.GenerateAggregateExpr()` produce expresiones aggregate correctas para cualquier campo enum, clasificando valores por keywords (terminal/negative/active/neutral) en EN y ES. La librería es reutilizada por init, migrate --split y fix.

## Criterios de Aceptacion (semanticos)

- [ ] GenerateAggregateExpr produce expresión correcta para enum EN (Completed, Blocked, In Progress, Pending)
- [ ] GenerateAggregateExpr produce expresión correcta para enum ES (Completado, Bloqueada, Diferida, En Progreso, Pending)
- [ ] GenerateAggregates skip campos no-enum y campos con aggregate existente

## Invariantes

- INV1: Tests existentes siguen pasando
  - Verificar: `go test ./... -race`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-implement-aggregate-generator.md) | Crear aggregate.go y aggregate_test.go con generador y 6 tests |

## Fuente de verdad

- `internal/rules/types.go` — SchemaField type definition
- `internal/migrate/` — package destino
- `docs/epics/E03-rootline/.stem` — ejemplo de expresión aggregate funcional
