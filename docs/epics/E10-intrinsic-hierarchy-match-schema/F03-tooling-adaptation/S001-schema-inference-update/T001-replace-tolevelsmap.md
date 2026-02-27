---
estado: Specified
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T001: Replace ToLevelsMap with match-based output

**Story**: [S001 Schema inference update](README.md)
**Contribuye a**: Inferred schemas preserve field distribution logic in v2 format

## Preserva

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
- INV5: Migration preserves field semantics exactly
  - Verificar: Generated v2 stem resolves identically for test records

## Contexto

`internal/infer/hierarchy.go` has `ToLevelsMap()` (lines 385-417) which converts `HierarchyResult` to the v1 `levels:` map structure. The analysis logic (`DetectLevels`, `distributeFields`) correctly separates root-level fields (present at all levels) from per-level fields (`OnlyHere`). This task replaces the output format: instead of generating `levels:`, generate v2 schema with inline `match:` on per-level fields.

## Especificacion Tecnica

In `internal/infer/hierarchy.go`:
- Add new method `ToMatchSchema() map[string]*SchemaField` on `HierarchyResult`
- Root fields (present at all levels via `distributeFields`) → no `Match` set
- Per-level fields (`OnlyHere`) → `Match` set to the level's glob pattern (e.g., `["E*"]`)
- Sequence id → `Match` as map: `{"E*": {prefix: "E", digits: 2}, ...}`
- Keep `ToLevelsMap()` temporarily (used by init command until T002 updates it)
- Add tests comparing `ToMatchSchema` output against expected v2 schema

## Dependencias

- F01 completed: SchemaField.Match type must exist

## Alcance

**In**:
1. Implement `ToMatchSchema()` method
2. Handle root fields (no match) and per-level fields (with match)
3. Handle sequence id map form
4. Add unit tests
5. Keep `ToLevelsMap()` for now

**Out**: Init command update (T002), migration (S002)

## Estado inicial esperado

- F01 completed: SchemaField has Match field with FieldMatch type
- `infer/hierarchy.go` has HierarchyResult with distributeFields, ToLevelsMap

## Criterios de Aceptacion

- `go test ./internal/infer/ -run TestToMatchSchema` passes
- Root-level fields have no Match
- Per-level fields have correct Match patterns
- Sequence id has map-form Match with correct prefix/digits per pattern
- `go test ./... -race` passes

## Fuente de verdad

- `internal/infer/hierarchy.go` — ToLevelsMap (lines 385-417), distributeFields, DetectLevels
- `internal/rules/rules.go` — SchemaField, FieldMatch type
