# S002: Drift detection testing

**Feature**: [F02 Automatic vertical consistency](../README.md)
**Capacidad**: Comprehensive test coverage for drift detection edge cases
**Cubre**: The testing/verification side of the F02 milestone

## Antes / Despues

**Antes**: No tests for vertical consistency — the concept doesn't exist in the engine yet.

**Despues**: Unit tests cover drift comparison logic (unanimous children mismatch, split children no-consensus, missing parent field, enum vs non-enum fields). E2E tests verify the full pipeline: real v2 `.stem` files + index + children → validate → drift warning in output.

## Criterios de Aceptacion (semanticos)

- [ ] Unit tests cover at least 5 distinct drift scenarios
- [ ] E2E tests use real temp directories with v2 .stem files
- [ ] Edge case: split children (no consensus) produces no warning
- [ ] Edge case: missing field on parent produces no warning

## Invariantes

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-unit-tests-drift-engine.md) | Unit tests for drift engine |
| [T002](T002-e2e-tests-drift-hierarchical.md) | E2E tests with real hierarchical stems |

## Fuente de verdad

- `internal/rules/` — Drift comparison functions (created in S001)
- `internal/e2e/` — E2E test patterns
