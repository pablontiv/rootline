---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Add PropagateAggregate type and detector with formula pre-check

**Story**: [S002 Propagate Detector](README.md)
**Contribuye a**: Stale index file produces PropagateAggregate proposal; formula pre-check prevents incorrect propagation

## Preserva

- INV1: Existing fix proposals unchanged
  - Verificar: `go test ./internal/proposal/ -race`
- INV3: Formula pre-check prevents incorrect propagation
  - Verificar: Test in T002

## Contexto

The gap between aggregate computation and disk write-back is the core problem. `AggregateAllSimple()` computes correct values in `record.Derived`. `ApplyProposals()` can write any `Proposal` to disk. What's missing is the detector that compares Derived vs Frontmatter and emits proposals.

The detector must also include a formula completeness pre-check: before emitting a proposal, verify that the aggregate formula references all distinct enum values present in the descendants. If any descendant value (like `Obsolete`) is not referenced in the formula string, skip propagation for that field and log a warning. This prevents writing `Pending` (the formula's default fallback) when the correct answer is `Obsolete`.

## Alcance

**In**:
1. Add `PropagateAggregate Type = "propagate_aggregate"` constant in `internal/proposal/proposal.go`
2. Add `PropagateAggregate int` field to `Summary` struct in `internal/proposal/proposal.go`
3. Create `internal/proposal/propagate.go` with `DetectPropagateAggregate(records []*extract.Record, effective *rules.StemFile) []Proposal`
4. For each index file (strings.HasSuffix README.md): compare `rec.Derived[field]` vs `rec.Frontmatter[field]` for each field in `effective.Aggregate`
5. Formula pre-check: extract distinct descendant values for the field; check each value appears as a quoted string in the aggregate expression; skip + warn if not
6. Emit `PropagateAggregate` proposal with From (stored) and To (computed)

**Out**: CLI wiring (S003), tests (T002), ApplyProposals case (S003)

## Estado inicial esperado

- `internal/proposal/proposal.go` has 12 existing type constants and Summary struct
- `internal/derive/aggregate.go` populates `record.Derived` via AggregateAllSimple
- `effective.Aggregate` is a `map[string]string` with field -> expression

## Criterios de Aceptacion

- `PropagateAggregate` type constant exists in proposal.go
- `PropagateAggregate` field exists in Summary struct
- `DetectPropagateAggregate` function compiles and returns `[]Proposal`
- Function skips non-index files
- Function skips fields where Derived matches Frontmatter
- Function skips fields where formula doesn't cover descendant values (pre-check)
- `go build ./...` compiles
- `go vet ./...` reports no issues

## Fuente de verdad

- `internal/proposal/proposal.go` — Type constants (L23-35), Summary (L59-72), Proposal struct (L38-48)
- `internal/derive/aggregate.go` — AggregateAllSimple, record.Derived population
- `internal/proposal/proposal.go:416` — detectInferFromChildren pattern (README.md suffix check)
