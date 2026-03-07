---
estado: Pending
tipo: historia
---
# S001: Stem-Health Formula Check

**Feature**: [F02 Formula Completeness Diagnostics](../README.md)
**Capacidad**: Stem-health check warns about incomplete aggregate formulas
**Cubre**: Proactive detection of formula gaps at schema validation time

## Antes / Despues

**Antes**: The aggregate formula for `estado` in `docs/epics/.stem` handles 5 values (Completed, Blocked, On Hold, In Progress, Specified). When all children are `Obsolete`, the formula falls through to `Pending` — semantically incorrect. No warning at schema level.

**Despues**: Stem-health check #8 compares aggregate expression string literals against enum values. If any enum value is not referenced in the expression, a warning is produced. Example: "aggregate formula for estado does not reference: Obsolete".

## Criterios de Aceptacion (semanticos)

- [ ] Formula covering all enum values produces no warning
- [ ] Formula missing `Obsolete` produces warning
- [ ] `go test ./internal/rules/ -race` passes

## Invariantes

- INV1: Existing 7 stem-health checks unchanged
  - Verificar: `go test ./internal/rules/ -run TestStemHealth -race`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-formula-completeness-check.md) | Add stem-health diagnostic for aggregate formula completeness |
| [T002](T002-formula-completeness-tests.md) | Unit tests: complete formula, incomplete formula, no enum field |

## Fuente de verdad

- `internal/rules/stem_health.go` — existing stem-health checks
- `docs/epics/.stem` — aggregate expressions (L60-66)
