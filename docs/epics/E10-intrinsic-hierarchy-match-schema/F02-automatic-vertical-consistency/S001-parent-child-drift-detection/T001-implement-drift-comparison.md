---
estado: Specified
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T001: Implement drift comparison logic

**Story**: [S001 Parent-child drift detection](README.md)
**Contribuye a**: Drift is detected for fields shared across levels

## Preserva

- INV1: All existing workflows produce identical results for v1 stems
  - Verificar: `go test ./... -race` passes

## Contexto

The Intrinsic Hierarchy Principle states that vertical consistency between parent and children is a property the engine must guarantee by default. Currently, no consistency check exists unless `aggregate:` is configured. This task implements the core drift comparison logic: given a parent record (index file) and its direct child records, compare shared fields (those without `match:` restriction) and detect when all children agree on a value that differs from the parent's value.

## Especificacion Tecnica

New file `internal/rules/drift.go` (or add to `validate.go`):
- `DriftWarning` struct: `Field`, `ParentValue`, `ChildrenValue`, `ParentPath`, `ChildPaths`
- `DetectDrift(parent Record, children []Record, schema map[string]*SchemaField) []DriftWarning`
- For each field in schema where `field.Match` is nil (applies everywhere):
  - Get parent's value for this field
  - Get all children's values for this field
  - If all children have the same value X and parent has value Y where X ≠ Y → DriftWarning
  - If children have mixed values → no warning (no consensus)
  - If parent doesn't have the field → no warning
  - If no children have the field → no warning

## Dependencias

- None directly, but assumes Record type from `internal/extract/` is available

## Alcance

**In**:
1. Define `DriftWarning` struct
2. Implement `DetectDrift` function
3. Handle edge cases: missing field, mixed children, empty children list
4. Unit tests for the comparison logic

**Out**: Integration with validation pipeline (T002), match-scoped exclusion (T003)

## Estado inicial esperado

- `internal/rules/validate.go` exists with `Validate` function
- Record type available from `internal/extract/`

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestDetectDrift` passes
- Unanimous children mismatch → DriftWarning returned
- Mixed children values → no warning
- Missing field on parent → no warning
- Empty children list → no warning
- `go vet ./...` passes

## Fuente de verdad

- `internal/rules/validate.go` — Validation engine (integration point)
- `internal/extract/` — Record type definition
- `docs/research/intrinsic-hierarchy-principle.md` — Part 3 (Automatic Vertical Consistency)
