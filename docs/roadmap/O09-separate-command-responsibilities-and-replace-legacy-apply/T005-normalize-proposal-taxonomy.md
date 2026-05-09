---
estado: Completed
tipo: task
---
# T005: Normalize proposal taxonomy

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE2 y CE3 del Outcome.

[[blocked_by:./T001-codify-command-responsibility-contracts.md]]
[[blocked_by:./T004-introduce-central-stem-resolution-api.md]]

## Preserva

- INV1: `.stem` mutation is explicit schema evolution, never a hidden side effect of data repair.
  - Verificar: proposal types declare target surface before any apply engine consumes them.

## Contexto

`AnalyzeReport` emits flat inferences, while `rootline/proposals` has richer proposal fields. Today the engines decide schema-vs-data behavior by ad-hoc type lists. A shared taxonomy is needed before implementing separate schema and repair apply paths.

## Alcance

**In**:
1. Define proposal surfaces: `schema`, `repair`, `bootstrap`, `migration`, `diagnostic`, and `requires_agent`.
2. Add or adapt JSON fields for target `.stem`, target document paths, operation id, confidence, patch preview, and skip reason.
3. Classify existing analyze inference types and fix proposal types.
4. Add tests for classification of `extend_enum`, `add_aggregate`, `remove_stem_field`, `correct_value`, `add_field`, `missing_schema`, `implicit_schema`, and agent-required categories.

**Out**:
- Applying proposals.
- Changing detector heuristics.

## Estado inicial esperado

- `internal/infer/report.go` and `internal/proposal/proposal.go` both exist but do not share an execution-ready taxonomy.

## Criterios de Aceptación

- A machine-readable proposal can be classified without inspecting command context.
- Schema-mutating proposals are distinguishable from document repairs.
- Analyzer outputs that are diagnostics or agent-required are not silently executable.
- Tests cover both analyze inferences and fix proposals.

## Fuente de verdad

- `internal/infer/report.go`
- `internal/infer/inference.go`
- `internal/proposal/proposal.go`
- `internal/proposal/stem_health.go`
- `cmd/rootline/analyze.go`
- `cmd/rootline/fix.go`
