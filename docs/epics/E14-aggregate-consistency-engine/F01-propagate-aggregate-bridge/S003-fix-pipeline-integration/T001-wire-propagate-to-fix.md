---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Wire propagation into fix CLI and apply pipeline

**Story**: [S003 Fix Pipeline Integration](README.md)
**Contribuye a**: fix --all propagates stale aggregate values; --no-propagate skips

## Preserva

- INV1: Existing proposals unchanged
  - Verificar: `go test ./internal/fix/ ./internal/proposal/ -race`

## Contexto

The detector (S002) produces `PropagateAggregate` proposals. This task wires them into the CLI and apply pipeline. Four integration points: (1) CLI flag `--no-propagate`, (2) helper function to append proposals, (3) `ApplyProposals` case in fix.go, (4) reporting case in `proposalsToFixResults`.

## Alcance

**In**:
1. `cmd/rootline/fix.go`: Add `fixNoPropagate bool` var and register `--no-propagate` flag in init()
2. `cmd/rootline/fix.go`: Add `appendPropagateProposals(report, records, effective)` helper function following the pattern of `appendAggregateProposals` (L248-255)
3. `cmd/rootline/fix.go`: Call `appendPropagateProposals` in `runFixAll` after L206, gated on `!fixNoPropagate`
4. `internal/fix/fix.go`: Add `case proposal.PropagateAggregate` in `ApplyProposals` switch — reuse `applyCorrectValue`
5. `cmd/rootline/fix.go`: Add `proposal.PropagateAggregate` case in `proposalsToFixResults` switch (L280) with format "propagate field: old -> new"

**Out**: E2E tests (T002), detector implementation (S002)

## Estado inicial esperado

- S002 completed: `DetectPropagateAggregate` function exists and is tested
- `cmd/rootline/fix.go` has `runFixAll` with derive pipeline (from S001)
- `internal/fix/fix.go` has `ApplyProposals` with existing switch cases

## Criterios de Aceptacion

- `rootline fix --all --dry-run docs/epics/` shows propagate_aggregate proposals
- `rootline fix --all --no-propagate --dry-run docs/epics/` shows no propagate proposals
- `go build ./cmd/rootline/` compiles
- `go test ./... -race` passes

## Fuente de verdad

- `cmd/rootline/fix.go` — runFixAll (L152-231), appendAggregateProposals (L248-255), proposalsToFixResults (L257-325)
- `internal/fix/fix.go` — ApplyProposals (L21-118), applyCorrectValue (L393-405)
- `internal/proposal/proposal.go` — PropagateAggregate type
