---
estado: Pending
tipo: feature
---
# F01: Propagate Aggregate Bridge

**Epic**: [E14 Aggregate Consistency Engine](../README.md)
**Satisface**: P1
**Objetivo**: Create the bridge between aggregate computation and disk write-back
**Beneficio**: Eliminates the #1 friction pattern — manual frontmatter propagation (26 sessions)
**Milestone**: `rootline fix --all` + `rootline validate --all` show 0 aggregate errors after propagation

## Scope

**In**: Derive pipeline in fix, PropagateAggregate detector with formula pre-check, CLI integration with default-on behavior
**Out**: Full formula completeness diagnostics (F02), post-merge hook (F03), derive: field write-back

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Derive Pipeline in Fix](S001-derive-pipeline-in-fix/) | fix --all runs aggregate computation before validation |
| S002 | [Propagate Detector](S002-propagate-detector/) | DetectPropagateAggregate emits proposals for stale index files |
| S003 | [Fix Pipeline Integration](S003-fix-pipeline-integration/) | fix --all propagates aggregate values by default |

## Invariantes

- INV1 (heredado): Existing fix proposal types continue working unchanged
  - Verificar: `go test ./internal/fix/ ./internal/proposal/ -race`
- INV2: Propagation only affects index files with aggregate definitions
- INV3: Formula pre-check prevents writing incorrect values

## Dependencias

- S001 → S002 → S003 (sequential)

## Fuente de verdad

- `cmd/rootline/fix.go` — runFixAll, appendAggregateProposals pattern
- `internal/derive/aggregate.go` — AggregateAllSimple
- `internal/proposal/proposal.go` — Proposal types, Analyze
- `internal/fix/fix.go` — ApplyProposals, applyCorrectValue
- `internal/rules/drift.go` — DriftWarning (NOT used, but documents the gap)
