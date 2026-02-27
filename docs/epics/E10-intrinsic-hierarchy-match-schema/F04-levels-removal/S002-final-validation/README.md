# S002: Final validation

**Feature**: [F04 Levels removal](../README.md)
**Capacidad**: Full regression confirms no breakage after levels removal
**Cubre**: The verification side of the F04 milestone

## Antes / Despues

**Antes**: After code cleanup in S001, the test suite may have broken tests referencing `levels:` or removed functions.

**Despues**: All tests pass. Coverage is ≥85%. Lint is clean. The codebase compiles and all CLI commands work correctly.

## Criterios de Aceptacion (semanticos)

- [ ] `go test ./... -race` passes with zero failures
- [ ] `golangci-lint run ./...` passes
- [ ] Code coverage ≥85%

## Invariantes

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
- INV3: MCP server behavior unchanged
  - Verificar: MCP tool tests pass

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-full-regression-suite.md) | Full regression test suite run |

## Fuente de verdad

- All test files in `internal/` and `cmd/`
