---
estado: Completed
tipo: test
ejecutable_en: 1 sesion
---
# T002: E2E tests with real hierarchical stems

**Story**: [S002 Drift detection testing](README.md)
**Contribuye a**: E2E tests use real temp directories with v2 .stem files

[[blocks:T002-integrate-drift-in-validation]]

## Preserva

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Integration tests need to verify the full pipeline: real v2 `.stem` files in a temp directory hierarchy with index files and children → `rootline validate` → drift warnings appear in output. These tests exercise the complete path from file scanning to drift detection to output formatting.

## Especificacion Tecnica

New tests in `internal/e2e/drift_test.go`:
1. Create temp directory with v2 `.stem`, README.md (index, estado: In Progress), and child dirs with README.md (estado: Completed) → validate → drift warning for estado
2. Create hierarchy where children have mixed estado values → validate → no drift warning
3. Create hierarchy with match-scoped field (tipo) → validate → no drift warning for tipo
4. Verify JSON output includes `drift_warnings` array
5. Verify table output shows drift section

## Dependencias

- F02/S001 completed: Full drift detection pipeline working

## Alcance

**In**:
1. E2E tests using temp directories with real .stem files
2. Test full validate pipeline with drift detection
3. Verify both JSON and table output formats
4. At least 3 distinct E2E scenarios

**Out**: Unit tests (T001), new validation features

## Estado inicial esperado

- F02/S001 completed: drift detection works end-to-end
- `internal/e2e/` directory exists with test patterns

## Criterios de Aceptacion

- `go test ./internal/e2e/ -run TestDrift` passes
- Tests create real temp dirs with v2 .stem files and markdown records
- Drift warning appears in validate output for matching scenario
- No false drift warnings for mixed-children or match-scoped fields
- `go test ./... -race` passes

## Fuente de verdad

- `internal/e2e/` — Existing E2E test patterns
- `internal/rules/validate.go` — ValidationResult with DriftWarnings
