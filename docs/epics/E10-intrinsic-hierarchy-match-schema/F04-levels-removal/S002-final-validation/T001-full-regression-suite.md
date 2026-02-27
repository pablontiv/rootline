---
estado: Specified
tipo: test
ejecutable_en: 1 sesion
---
# T001: Full regression test suite run

**Story**: [S002 Final validation](README.md)
**Contribuye a**: All tests pass, coverage ≥85%, lint clean

[[blocks:T003-remove-checknesting-from-cli]]
[[blocks:T002-remove-stemhealth-checks-and-mergelevels]]

## Preserva

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
- INV3: MCP server behavior unchanged
  - Verificar: `go test ./internal/mcp/ -race` passes

## Contexto

After all levels removal tasks (S001), the test suite needs a comprehensive verification pass. Some tests that reference `levels:` may have been missed during cleanup, and coverage may have dropped if v1-specific test code was deleted without replacement.

## Especificacion Tecnica

1. Run `go test ./... -race` — fix any compilation errors or test failures
2. Run `golangci-lint run ./...` — fix any lint issues
3. Check coverage: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
4. If coverage < 85%: identify uncovered code and add tests
5. Run `rootline validate --all docs/epics/` — verify project dogfooding works
6. Verify all CLI commands work: validate, describe, query, tree, stats, fix, graph, init

## Dependencias

- S001 completed: all levels code removed

## Alcance

**In**:
1. Run full test suite and fix failures
2. Run lint and fix issues
3. Verify coverage threshold
4. Run rootline CLI commands for smoke testing
5. Add tests if coverage dropped

**Out**: New features, documentation

## Estado inicial esperado

- S001 completed: all levels code removed
- Some tests may reference removed code

## Criterios de Aceptacion

- `go test ./... -race` passes with zero failures
- `golangci-lint run ./...` passes with zero issues
- Code coverage ≥ 85%
- `rootline validate --all docs/epics/` succeeds
- `rootline describe docs/epics/` shows correct output

## Fuente de verdad

- All test files in `internal/` and `cmd/`
- `docs/epics/.stem` — v2 stem being used by the project
