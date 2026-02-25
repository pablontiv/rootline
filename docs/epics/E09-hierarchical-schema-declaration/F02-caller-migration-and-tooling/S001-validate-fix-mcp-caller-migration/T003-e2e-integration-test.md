---
estado: Specified
tipo: software-test
ejecutable_en: 1 sesion
---
# T003: E2E integration test with levels fixture

**Story**: [S001 Validate/Fix/MCP Caller Migration](README.md)
**Contribuye a**: E2E test con fixture de levels verifica pipeline completo

[[blocks:T002-migrate-fix-and-mcp-callers]]

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV2: Coverage >= 85%
  - Verificar: `go test ./... -coverprofile=c.out && go tool cover -func=c.out | tail -1`

## Contexto

An end-to-end test that creates a temporary directory hierarchy with a `.stem` file using `levels:`, populates it with documents at various nesting levels (some valid, some invalid), and runs the full validate pipeline to verify correct behavior.

Existing E2E tests are in `internal/e2e/` and use Go's testing package with temporary directories.

The fixture should test:
1. Documents at correct levels validate successfully with per-level schema
2. Documents at incorrect nesting are reported as errors
3. Documents without levels in stem work unchanged

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
paquete: internal/e2e
cobertura_objetivo: ">= 85%"
tipos_test:
  - e2e
fixtures:
  - Temp directory with .stem containing levels (4 levels: epic/feature/story/task)
  - Valid docs at each level
  - Invalid doc at wrong nesting level (task under epic)
  - Comparison .stem without levels (backward compat)
```

## Dependencias

- T002: All callers migrated to ResolveForRecord

## Alcance

**In**:
1. Create `internal/e2e/hierarchy_test.go` (new file)
2. TestHierarchyLevels: full pipeline with levels fixture
3. TestHierarchyLevelsNesting: nesting violation detected
4. TestHierarchyLevelsBackwardCompat: without levels, same behavior as before
5. Use `t.TempDir()` for test fixtures

**Out**: Unit tests (already in F01), performance benchmarks

## Estado inicial esperado

- All F01 and F02/S001/T001-T002 work complete
- `internal/e2e/` directory exists with existing E2E tests as patterns

## Criterios de Aceptacion

- `go test ./internal/e2e/ -run TestHierarchy -v` passes
- Test verifies per-level schema enforcement (task gets `ejecutable_en: required`, feature doesn't)
- Test verifies nesting violation is detected
- Test verifies backward compat (no levels = same as before)
- `go test ./... -race` passes
- Coverage >= 85%

## Fuente de verdad

- `internal/e2e/` — existing E2E test patterns
- `internal/rules/hierarchy.go` — ResolveForRecord, ExpandLevels, CheckNesting
