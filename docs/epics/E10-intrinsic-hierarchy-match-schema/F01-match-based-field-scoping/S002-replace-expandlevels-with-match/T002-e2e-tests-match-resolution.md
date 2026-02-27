---
estado: Completed
tipo: test
ejecutable_en: 1 sesion
---
# T002: E2E tests for match-based resolution

**Story**: [S002 Replace ExpandLevels with match resolution](README.md)
**Contribuye a**: E2E tests covering the full validate/describe/fix pipeline pass with v2 stems

[[blocks:T001-rewrite-resolveforrecord]]

## Preserva

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

`internal/e2e/hierarchy_test.go` has comprehensive tests for the v1 levels-based resolution pipeline. These tests verify per-level schema, nesting validation, sequence numbering, and describe output using v1 `.stem` files with `levels:`. This task creates parallel E2E tests that use v2 `.stem` files with `match:` to verify identical behavior through the full pipeline.

## Especificacion Tecnica

New test file `internal/e2e/match_hierarchy_test.go` (or extend `hierarchy_test.go`):
- Create temp directory structures with v2 `.stem` files using `match:` fields
- Test cases mapping to existing v1 tests: per-level fields, sequence numbering, describe output
- Verify `rootline validate`, `rootline describe`, and `rootline fix` produce correct results
- Compare v2 output against expected values (not against v1 output — independent verification)

## Dependencias

- T001 (S002): ResolveForRecord must support v2

## Alcance

**In**:
1. E2E tests for v2 match-based resolution
2. Test per-level field filtering (tipo only at F* and T*)
3. Test sequence id with per-pattern prefix/digits
4. Test describe output for v2 stems
5. Test validate with v2 stems

**Out**: Drift detection tests (F02), migration tests (F03)

## Estado inicial esperado

- S002/T001 completed: ResolveForRecord supports v2
- `internal/e2e/hierarchy_test.go` exists as reference for test patterns

## Criterios de Aceptacion

- `go test ./internal/e2e/ -run TestMatchHierarchy` passes
- Tests cover: per-level field filtering, sequence numbering, describe output, validate with v2 stems
- At least 5 test cases covering distinct v2 resolution scenarios
- `go test ./... -race` passes

## Fuente de verdad

- `internal/e2e/hierarchy_test.go` — Existing E2E tests (reference for patterns)
- `internal/rules/hierarchy.go` — ResolveForRecord with v2 support
