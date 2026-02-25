---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Add stemhealth checks for levels validity

**Story**: [S002 Describe, Infer and Stemhealth](README.md)
**Contribuye a**: stemhealth detecta levels malformados

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`

## Contexto

`internal/rules/stemhealth.go` implements 7 health checks for `.stem` files (yaml-valid, scope-match, type-consistency, enum-values, rule-field-exists, field-override, aggregated-required). These run as a pre-phase during `validate --all`.

With `levels:` support, two new health checks are needed:
1. **levels-children-valid**: Each level's `children` list references level names that actually exist in the same levels map
2. **levels-no-cycles**: The children graph has no cycles (e.g., epic -> feature -> epic)

These checks validate the `.stem` file itself, not individual records (that's CheckNesting's job).

## Dependencias

- F01/S001/T001: HierarchyLevel struct must exist

## Alcance

**In**:
1. Add `checkLevelsChildrenValid` function to `internal/rules/stemhealth.go`
2. Add `checkLevelsNoCycles` function (DFS cycle detection on children graph)
3. Register both checks in the health check list
4. Add tests in `internal/rules/stemhealth_test.go`
5. Verify existing stemhealth tests pass

**Out**: infer updates (T002), record-level nesting (F01/S002)

## Estado inicial esperado

- `internal/rules/stemhealth.go` exists with 7 health checks
- HierarchyLevel struct exists with Children field
- `.stem` files can have Levels field

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestStemHealth -v` passes including new checks
- Levels with children referencing non-existent level names → error
- Levels with circular children → error
- Levels without cycles → passes
- `.stem` without levels → checks are skipped (no errors)
- `go test ./... -race` passes

## Fuente de verdad

- `internal/rules/stemhealth.go` — existing health checks
- `internal/doctor/` — doctor package if health checks are registered there
