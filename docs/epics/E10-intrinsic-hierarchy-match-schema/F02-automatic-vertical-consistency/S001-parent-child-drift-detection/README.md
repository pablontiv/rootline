# S001: Parent-child drift detection

**Feature**: [F02 Automatic vertical consistency](../README.md)
**Capacidad**: Engine detects field drift between parent index and children without requiring aggregate expressions
**Cubre**: The detection logic and validation integration side of the F02 milestone

## Antes / Despues

**Antes**: No consistency check between parent and child unless `aggregate:` is explicitly configured. A parent README with `estado: In Progress` while all children have `estado: Completed` goes undetected. The only way to get consistency is manual configuration of `aggregate:` expressions.

**Despues**: For any field without `match:` restriction (meaning it applies at all levels), if all direct children of a parent have the same value X and the parent has value Y where Y ≠ X, validation produces a warning. Fields with `match:` restrictions are excluded from drift checks since they don't exist at all levels.

## Criterios de Aceptacion (semanticos)

- [ ] Drift is detected for fields shared across levels (no `match:` restriction)
- [ ] Drift warnings appear in both table and JSON validation output
- [ ] Match-scoped fields are excluded from drift checks
- [ ] Drift is a warning, not an error — it doesn't fail validation

## Invariantes

- INV1: All existing workflows produce identical results for v1 stems
  - Verificar: `go test ./... -race` passes
- INV4: Drift warnings are warnings, not errors
  - Verificar: Validation exit code is 0 when only drift warnings exist

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-implement-drift-comparison.md) | Implement drift comparison logic |
| [T002](T002-integrate-drift-in-validation.md) | Integrate drift warnings in validation output |
| [T003](T003-handle-match-scoped-exclusion.md) | Handle match-scoped fields exclusion |

## Fuente de verdad

- `internal/rules/validate.go` — Validation engine
- `internal/derive/aggregate.go` — Existing aggregation (reference)
- `internal/index/` — Directory scanner, file indexing
