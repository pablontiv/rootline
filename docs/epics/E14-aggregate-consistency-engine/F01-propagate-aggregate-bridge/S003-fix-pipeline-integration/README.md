---
estado: Pending
tipo: historia
---
# S003: Fix Pipeline Integration

**Feature**: [F01 Propagate Aggregate Bridge](../README.md)
**Capacidad**: fix --all propagates aggregate values by default with --no-propagate to disable
**Cubre**: End-to-end working feature

## Antes / Despues

**Antes**: `fix --all` cannot resolve aggregate drift. Users must manually edit 3+ files per completed task. The #1 friction pattern across 200+ sessions (26 with evidence).

**Despues**: `fix --all` propagates aggregate values by default. `--no-propagate` disables. Full write-back to disk via existing `ApplyProposals` + `RewriteFrontmatter` infrastructure.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline fix --all docs/epics/` propagates stale aggregate values
- [ ] `rootline fix --all --no-propagate docs/epics/` skips propagation
- [ ] `rootline validate --all docs/epics/` shows 0 aggregate errors after fix
- [ ] Deep hierarchy (3 levels) propagates bottom-up correctly

## Invariantes

- INV1: Existing proposals unchanged
  - Verificar: `go test ./internal/fix/ ./internal/proposal/ -race`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-wire-propagate-to-fix.md) | Add --no-propagate flag, appendPropagateProposals, ApplyProposals case, reporting |
| [T002](T002-e2e-propagate-test.md) | E2E round-trip: stale README to fix to verify corrected; deep hierarchy |

## Fuente de verdad

- `cmd/rootline/fix.go` — runFixAll, appendAggregateProposals pattern (L248-255), proposalsToFixResults (L257)
- `internal/fix/fix.go` — ApplyProposals switch (L52-113), applyCorrectValue (L393-405)
