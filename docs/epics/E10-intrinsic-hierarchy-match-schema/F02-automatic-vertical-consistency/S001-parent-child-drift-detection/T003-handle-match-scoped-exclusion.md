---
estado: Specified
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T003: Handle match-scoped fields exclusion

**Story**: [S001 Parent-child drift detection](README.md)
**Contribuye a**: Match-scoped fields are excluded from drift checks

[[blocks:T001-implement-drift-comparison]]

## Preserva

- INV1: All existing workflows produce identical results for v1 stems
  - Verificar: `go test ./... -race` passes

## Contexto

Fields with `match:` restrictions only apply to specific directory patterns (e.g., `tipo` only at `F*` and `T*` levels). These fields should NOT participate in drift detection because they don't exist at all levels — comparing a parent that doesn't have `tipo` against children that do is meaningless. The `DetectDrift` function already skips fields with `Match != nil`, but this task ensures the logic is correct and comprehensive, including edge cases like match-scoped fields that happen to exist at both parent and child levels.

## Especificacion Tecnica

Review and extend `DetectDrift` in `internal/rules/drift.go`:
- Confirm: fields with `field.Match != nil` are excluded from drift detection
- Edge case: if a match-scoped field happens to exist on both parent and child (because both match the pattern), it should still be excluded — match-scoped means "not universal", so drift detection doesn't apply
- Add explicit tests for match-scoped field exclusion

## Dependencias

- T001: DetectDrift function must exist

## Alcance

**In**:
1. Verify match-scoped field exclusion logic in DetectDrift
2. Add tests for edge cases: match-scoped field present at both levels, match-scoped field missing at parent
3. Document the exclusion behavior

**Out**: Integration with validation (T002), new match patterns

## Estado inicial esperado

- T001 completed: DetectDrift exists with basic match exclusion
- SchemaField.Match is populated correctly

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestDriftMatchExclusion` passes
- Match-scoped fields never produce drift warnings regardless of values
- Fields without Match produce drift warnings when appropriate

## Fuente de verdad

- `internal/rules/drift.go` — DetectDrift function (from T001)
- `internal/rules/rules.go` — SchemaField.Match
