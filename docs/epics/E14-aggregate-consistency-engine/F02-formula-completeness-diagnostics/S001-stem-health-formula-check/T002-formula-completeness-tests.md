---
estado: Completed
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Unit tests for formula completeness check

**Story**: [S001 Stem-Health Formula Check](README.md)
**Contribuye a**: Formula covering all enum values produces no warning

[[blocks:T001-formula-completeness-check]]

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`

## Contexto

The formula completeness stem-health check needs tests for: complete formulas (all enum values covered), incomplete formulas (missing values), fields without enums, and non-aggregate fields. Tests should follow the existing stem-health test patterns.

## Alcance

**In**:
1. Add tests in `internal/rules/stem_health_test.go`
2. Test cases:
   - Complete formula: all 5 enum values referenced -> 0 warnings
   - Incomplete formula: missing `Obsolete` -> 1 warning mentioning `Obsolete`
   - No enum field: aggregate on a non-enum field -> 0 warnings (check skipped)
   - No aggregate: stem without aggregate section -> 0 warnings

**Out**: Integration with validate command, formula auto-fix

## Estado inicial esperado

- T001 completed: formula completeness check exists
- `internal/rules/stem_health_test.go` exists with existing test patterns

## Criterios de Aceptacion

- 4 test cases pass
- `go test ./internal/rules/ -run TestStemHealth_FormulaCompleteness -race -v` passes

## Fuente de verdad

- `internal/rules/stem_health.go` — formula completeness check
- `internal/rules/stem_health_test.go` — existing test patterns
