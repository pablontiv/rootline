---
estado: Completed
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T001: Rewrite ResolveForRecord for match model

**Story**: [S002 Replace ExpandLevels with match resolution](README.md)
**Contribuye a**: ResolveForRecord uses match filtering instead of virtual stems

[[blocks:T002-implement-match-aware-field-resolution]]
[[blocks:T003-add-v2-stem-parsing]]

## Preserva

- INV1: All existing workflows produce identical results for v1 stems
  - Verificar: `go test ./... -race` passes

## Contexto

`ResolveForRecord` in `internal/rules/hierarchy.go` (lines 92-104) currently calls `ExpandLevels` when `merged.Levels != nil` to create virtual stem entries from the levels map. For v2 stems, this expansion is unnecessary — fields already carry their `Match` metadata. Instead, the merged schema should be filtered by `FilterSchemaByMatch` to produce the effective schema for the record's path.

## Especificacion Tecnica

Modify `ResolveForRecord` in `internal/rules/hierarchy.go`:
- After `MergeStemFiles(entries)`, check `merged.Version`
- If v2: call `FilterSchemaByMatch(merged.Schema, recordPath)` and set the filtered result as the effective schema
- If v1: call `ExpandLevels` as before (backward compat)
- Ensure the returned `*StemFile` has the correct effective schema for both paths
- The v2 path should produce identical results to v1 for equivalent configurations

## Dependencias

- T002 (S001): FilterSchemaByMatch function must exist
- T003 (S001): v2 stem parsing must work

## Alcance

**In**:
1. Add v2 branch in ResolveForRecord
2. Call FilterSchemaByMatch for v2 stems
3. Keep v1 branch unchanged
4. Add unit tests comparing v1 and v2 resolution for equivalent stems

**Out**: Removing the v1 path (F04), E2E tests (T002)

## Estado inicial esperado

- S001 completed: SchemaField.Match, FilterSchemaByMatch, v2 parsing all working
- `ResolveForRecord` exists with v1 logic

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestResolveForRecord` passes — both v1 and v2 paths
- v2 resolution produces identical effective schema to v1 for equivalent configurations
- Describe output is identical for v1 and v2 equivalent stems
- `go test ./... -race` passes (full suite)

## Fuente de verdad

- `internal/rules/hierarchy.go` — ResolveForRecord (lines 92-104), ExpandLevels (lines 56-86)
