---
estado: Completed
tipo: historia
---
# S001: Derive Pipeline in Fix

**Feature**: [F01 Propagate Aggregate Bridge](../README.md)
**Capacidad**: fix --all runs aggregate computation before validation
**Cubre**: Fix command can detect aggregate mismatches (prerequisite for propagation)

## Antes / Despues

**Antes**: `fix --all` never runs the derive pipeline. `record.Derived` is always empty during fix. Aggregate errors are invisible to the fix command — it cannot detect or report stale aggregate values.

**Despues**: `fix --all` runs `DeriveAllSimple` + `EnrichBuiltinsSimple` + `AggregateAllSimple` before validation. Aggregate errors appear in fix output. This is the prerequisite for the propagation detector.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline fix --all --dry-run docs/epics/` shows aggregate errors in output when parent README has stale estado
- [ ] `go test ./... -race` passes

## Invariantes

- INV1: Existing fix proposals unchanged
  - Verificar: `go test ./internal/fix/ -race`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-derive-pipeline.md) | Add derive pipeline calls to runFixAll |
| [T002](T002-verify-derive-in-fix.md) | Test that fix --all detects aggregate errors |

## Fuente de verdad

- `cmd/rootline/fix.go` — runFixAll (L152-231)
- `internal/derive/aggregate.go` — AggregateAllSimple (L160-163)
- `internal/derive/record.go` — DeriveAllSimple, EnrichBuiltinsSimple
