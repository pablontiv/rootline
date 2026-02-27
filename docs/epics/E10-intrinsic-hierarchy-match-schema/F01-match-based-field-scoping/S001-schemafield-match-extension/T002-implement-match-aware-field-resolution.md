---
estado: Specified
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T002: Implement match-aware field resolution

**Story**: [S001 SchemaField match extension](README.md)
**Contribuye a**: v2 `.stem` with `match:` on fields parses without error and produces correct SchemaField.Match values

[[blocks:T001-add-match-field-to-schemafield]]

## Preserva

- INV1: All existing workflows produce identical results for v1 stems
  - Verificar: `go test ./... -race` passes

## Contexto

With `SchemaField.Match` in place (T001), we need a function that filters schema fields based on a record's path. Given a record at `docs/epics/E01-name/F01-name/S001-name/T001-task.md`, the function should check each path component against each field's `Match` patterns and return only the fields that apply to this record. Fields without `Match` apply everywhere. Fields with `Match` apply only when at least one path component matches a pattern.

## Especificacion Tecnica

New function in `internal/rules/` (likely `hierarchy.go` or new file `match.go`):
- `FilterSchemaByMatch(schema map[string]*SchemaField, recordPath string) map[string]*SchemaField`
- For each field: if `field.Match` is nil → include (applies everywhere)
- If `field.Match` has patterns → check if any path component of `recordPath` matches any pattern via `filepath.Match`
- For map-form match (sequence id): resolve the specific config for the matching pattern
- Handle `RequiredMatch`: if field has `required: true` but `RequiredMatch` restricts it, set effective `required` based on path match

## Dependencias

- T001: SchemaField must have the Match field

## Alcance

**In**:
1. Implement `FilterSchemaByMatch` function
2. Handle all three Match forms (string, list, map)
3. Handle RequiredMatch scoping
4. Unit tests for filtering with various path/pattern combinations

**Out**: Integration with ResolveForRecord (S002/T001), v2 parsing (T003)

## Estado inicial esperado

- T001 completed: `SchemaField` has `Match` field with `FieldMatch` type
- `filepath.Match` available in Go stdlib

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestFilterSchemaByMatch` passes — tests filtering for paths at different depths
- Field without Match applies at all levels
- Field with `match: ["F*", "T*"]` applies only when path has F* or T* components
- Field with map-form match resolves correct config per pattern
- RequiredMatch correctly scopes the required flag

## Fuente de verdad

- `internal/rules/rules.go` — SchemaField with Match (from T001)
- `internal/rules/hierarchy.go` — existing ExpandLevels for behavior reference
