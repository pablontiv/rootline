---
estado: Specified
tipo: refactor
ejecutable_en: 1 sesion
---
# T002: Remove stemhealth checks 8-9 and mergeLevels

**Story**: [S001 Code cleanup](README.md)
**Contribuye a**: All levels-related code removed

[[blocks:T001-remove-hierarchylevel-and-expandlevels]]

## Preserva

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

`internal/rules/stemhealth.go` has checks 8 (`levels-children-valid`) and 9 (`levels-no-cycles`) that validate `levels:` declarations. `internal/rules/merge.go` has `mergeLevels` function for merging `Levels` maps during stem merge. With `HierarchyLevel` removed (T001), these are dead code that won't compile.

## Especificacion Tecnica

In `internal/rules/stemhealth.go`:
- Delete check 8 (`levels-children-valid`, lines ~253-300)
- Delete check 9 (`levels-no-cycles`, lines ~300-350)
- Update the check list/registry to skip removed checks
- Update stemhealth tests in `stemhealth_test.go`

In `internal/rules/merge.go`:
- Delete `mergeLevels` function (lines 204-223)
- Remove call to `mergeLevels` from the main merge function
- Update merge tests in `merge_test.go`

## Dependencias

- T001: HierarchyLevel removed first (these reference it)

## Alcance

**In**:
1. Delete stemhealth checks 8 and 9
2. Delete mergeLevels function
3. Remove mergeLevels call from merge pipeline
4. Update/delete affected tests

**Out**: CLI call sites (T003)

## Estado inicial esperado

- T001 completed: HierarchyLevel struct deleted

## Criterios de Aceptacion

- `grep -r "mergeLevels\|levels-children-valid\|levels-no-cycles" internal/rules/` returns zero matches
- `go build ./cmd/rootline/` succeeds
- `go test ./internal/rules/ -race` passes
- Stemhealth check count decreased by 2

## Fuente de verdad

- `internal/rules/stemhealth.go` — Checks 8-9 (lines ~253-350)
- `internal/rules/merge.go` — mergeLevels (lines 204-223)
