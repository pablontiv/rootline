---
estado: Completed
tipo: test
ejecutable_en: 1 sesion
---
# T001: Unit tests for drift engine

**Story**: [S002 Drift detection testing](README.md)
**Contribuye a**: Unit tests cover at least 5 distinct drift scenarios

[[blocks:T003-handle-match-scoped-exclusion]]

## Preserva

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

The drift detection engine (from F02/S001) needs comprehensive unit testing beyond what was done in the implementation tasks. This task focuses on edge cases and scenario completeness to ensure the drift engine is robust.

## Especificacion Tecnica

Add tests to `internal/rules/drift_test.go`:
1. Unanimous children mismatch with parent → warning
2. Split children (mixed values) → no warning (no consensus)
3. Parent field missing → no warning
4. All children field missing → no warning
5. Single child matches parent → no warning
6. Single child differs from parent → warning
7. Empty children list → no warning
8. Enum field drift vs non-enum field drift → both detected
9. Multiple fields with drift → multiple warnings
10. Match-scoped field → excluded (verified separately)

## Dependencias

- F02/S001 completed: DetectDrift, drift integration, match exclusion all implemented

## Alcance

**In**:
1. Comprehensive unit tests for DetectDrift
2. At least 10 test cases covering distinct scenarios
3. Table-driven tests for maintainability

**Out**: E2E tests (T002), new drift features

## Estado inicial esperado

- `internal/rules/drift.go` exists with DetectDrift function
- Basic tests from S001 tasks exist

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestDetectDrift` passes with ≥10 test cases
- Each test case has a descriptive name
- Edge cases (empty list, missing field, single child) all covered
- `go test ./... -race` passes

## Fuente de verdad

- `internal/rules/drift.go` — DetectDrift function
