# S002: Replace ExpandLevels with match resolution

**Feature**: [F01 Match-based field scoping](../README.md)
**Capacidad**: ResolveForRecord uses match filtering instead of virtual stem entries from ExpandLevels
**Cubre**: The resolution engine side of the F01 milestone — fields are resolved by matching path components

## Antes / Despues

**Antes**: `ResolveForRecord` calls `ExpandLevels` which iterates path components, matches them against `levels:` entries, creates virtual `StemFile` entries with per-level schema, and appends them for re-merge. This is the only consumption point for the `levels:` data structure.

**Despues**: For v2 stems, `ResolveForRecord` applies match-based field filtering directly on the merged schema — iterating `Schema` fields and keeping only those whose `Match` patterns match the record's path components. No virtual stem entries, no `ExpandLevels` call. v1 stems still use the old path.

## Criterios de Aceptacion (semanticos)

- [ ] v2 stem resolution produces identical effective schema as v1 for equivalent configurations
- [ ] Per-level fields (e.g., `tipo` only at F* and T* levels) are correctly filtered
- [ ] Sequence id with per-pattern prefix/digits resolves correctly at each level
- [ ] E2E tests covering the full validate/describe/fix pipeline pass with v2 stems

## Invariantes

- INV1: All existing workflows produce identical results for v1 stems
  - Verificar: `go test ./... -race` passes
- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-rewrite-resolveforrecord.md) | Rewrite ResolveForRecord for match model |
| [T002](T002-e2e-tests-match-resolution.md) | E2E tests for match-based resolution |

## Fuente de verdad

- `internal/rules/hierarchy.go` — ResolveForRecord, ExpandLevels
- `internal/e2e/hierarchy_test.go` — Existing E2E tests to port
