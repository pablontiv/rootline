---
estado: Completed
tipo: refactor
ejecutable_en: 1 sesion
---
# T003: Remove CheckNesting from CLI call sites

**Story**: [S001 Code cleanup](README.md)
**Contribuye a**: All levels-related code removed

[[blocks:T001-remove-hierarchylevel-and-expandlevels]]

## Preserva

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
- INV3: MCP server behavior unchanged
  - Verificar: `go test ./internal/mcp/ -race` passes

## Contexto

`CheckNesting` is called from CLI commands to enforce that directory nesting follows the `children:` declarations in `levels:`. Under the intrinsic hierarchy model, every indexed subdirectory is a valid child — there's nothing to check. The calls exist in `cmd/rootline/validate.go` (2 places, lines ~102-107 and ~182-185) and `cmd/rootline/fix.go` (1 place, line ~112-116).

## Especificacion Tecnica

In `cmd/rootline/validate.go`:
- Remove both `CheckNesting` call sites
- Remove any nesting-related error collection and output

In `cmd/rootline/fix.go`:
- Remove `CheckNesting` call site

In `internal/rules/hierarchy.go`:
- Delete `CheckNesting` function (if not already deleted in T001)

Update tests:
- Remove nesting violation test cases from `internal/e2e/hierarchy_test.go`
- Update CLI command tests if they expect nesting errors

## Dependencias

- T001: HierarchyLevel and ExpandLevels removed (CheckNesting references them)

## Alcance

**In**:
1. Remove CheckNesting calls from validate.go (2 places)
2. Remove CheckNesting call from fix.go (1 place)
3. Delete CheckNesting function
4. Update/remove nesting violation tests

**Out**: Other CLI changes, new validation features

## Estado inicial esperado

- T001 completed: HierarchyLevel deleted
- CheckNesting function and call sites exist

## Criterios de Aceptacion

- `grep -r "CheckNesting" cmd/ internal/` returns zero matches
- `go build ./cmd/rootline/` succeeds
- `go test ./... -race` passes
- No nesting-related errors in `rootline validate` output

## Fuente de verdad

- `cmd/rootline/validate.go` — CheckNesting calls (lines ~102-107, ~182-185)
- `cmd/rootline/fix.go` — CheckNesting call (line ~112-116)
- `internal/rules/hierarchy.go` — CheckNesting function (lines 12-49)
