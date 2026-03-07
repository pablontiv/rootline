---
estado: Completed
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Unit tests for propagate detector

**Story**: [S002 Propagate Detector](README.md)
**Contribuye a**: Formula pre-check prevents incorrect propagation; stale index produces proposal

[[blocks:T001-propagate-type-and-detector]]

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`

## Contexto

`DetectPropagateAggregate` needs comprehensive unit tests covering all edge cases. The function takes records (with populated Derived fields) and an effective stem (with Aggregate definitions), and returns proposals. Tests must verify correct behavior for: stale values, consistent values, missing frontmatter, non-index files, stems without aggregates, mixed staleness, and the formula pre-check guard.

## Alcance

**In**:
1. Create `internal/proposal/propagate_test.go`
2. Test cases:
   - Stale value: index file with `Frontmatter["estado"]="Pending"`, `Derived["estado"]="Completed"` -> 1 proposal with From="Pending", To="Completed"
   - Already consistent: Frontmatter matches Derived -> 0 proposals
   - Missing frontmatter field: no `estado` in Frontmatter -> 0 proposals (handled by AddField)
   - Non-index file: T001.md with stale value -> 0 proposals
   - No aggregate in stem: effective.Aggregate is nil -> 0 proposals
   - Mixed staleness: multiple fields, some stale some consistent -> proposals only for stale ones
   - Uncovered enum value: descendant has `Obsolete` but formula doesn't reference it -> 0 proposals + verify no proposal emitted

**Out**: E2E tests (S003), CLI integration tests

## Estado inicial esperado

- T001 completed: DetectPropagateAggregate function exists
- `internal/proposal/` has existing test files for pattern reference

## Criterios de Aceptacion

- 7 test cases pass
- `go test ./internal/proposal/ -run TestDetectPropagateAggregate -race -v` passes
- Tests cover: stale, consistent, missing, non-index, no-aggregate, mixed, uncovered-enum

## Fuente de verdad

- `internal/proposal/propagate.go` — DetectPropagateAggregate function
- `internal/proposal/proposal_test.go` — existing test patterns
