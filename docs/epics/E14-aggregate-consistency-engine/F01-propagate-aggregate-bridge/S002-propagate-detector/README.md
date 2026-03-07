---
estado: Completed
tipo: historia
---
# S002: Propagate Detector

**Feature**: [F01 Propagate Aggregate Bridge](../README.md)
**Capacidad**: DetectPropagateAggregate emits proposals for stale index files with formula pre-check
**Cubre**: Core detection logic with safety guard

## Antes / Despues

**Antes**: No mechanism converts aggregate mismatches into fixable proposals. DriftWarning and Proposal are separate pipelines with no bridge. The computed value lives in `record.Derived` but is never compared against `record.Frontmatter` for proposal generation.

**Despues**: `DetectPropagateAggregate` compares `record.Derived` vs `record.Frontmatter` for index files and emits `PropagateAggregate` proposals. A formula completeness pre-check prevents writing wrong values when the formula doesn't cover all descendant enum values (e.g., Obsolete).

## Criterios de Aceptacion (semanticos)

- [ ] Stale index file produces PropagateAggregate proposal with correct From/To
- [ ] Formula doesn't cover descendant value: no proposal emitted, warning logged
- [ ] `go test ./internal/proposal/ -race` passes

## Invariantes

- INV2: Only index files with aggregate definitions affected
  - Verificar: Non-index files remain unmodified in tests
- INV3: Formula pre-check prevents incorrect propagation
  - Verificar: Test with Obsolete children produces 0 proposals

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-propagate-type-and-detector.md) | Add PropagateAggregate type and DetectPropagateAggregate with formula pre-check |
| [T002](T002-propagate-detector-tests.md) | Unit tests: stale, consistent, missing, non-index, no aggregate, uncovered enum |

## Fuente de verdad

- `internal/proposal/proposal.go` — Proposal type, Type constants, Summary struct
- `internal/derive/aggregate.go` — AggregateAllSimple, record.Derived population
- `internal/rules/drift.go` — DriftWarning (documents the gap, not used directly)
