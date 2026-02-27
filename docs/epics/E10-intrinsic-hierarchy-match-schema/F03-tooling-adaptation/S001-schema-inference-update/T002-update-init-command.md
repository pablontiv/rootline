---
estado: Completed
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T002: Update init command to generate v2 stems

**Story**: [S001 Schema inference update](README.md)
**Contribuye a**: `rootline init` on a hierarchical directory produces a v2 .stem with match fields

[[blocks:T001-replace-tolevelsmap]]

## Preserva

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

`cmd/rootline/init.go` has `buildHierarchicalStems` (line ~160) and `generateHierarchicalRootYAML` (line ~200) which produce v1 `.stem` files with `levels:` sections when hierarchy is detected. With `ToMatchSchema()` available (T001), these functions should generate v2 stems with `version: 2` and inline `match:` on fields instead of `levels:`.

## Especificacion Tecnica

Modify `cmd/rootline/init.go`:
- Replace calls to `ToLevelsMap()` with `ToMatchSchema()`
- Update `generateHierarchicalRootYAML` to produce v2 format: `version: 2`, schema fields with inline match, no `levels:` section
- Update `buildHierarchicalStems` to use v2 output
- Update `init_test.go` to expect v2 format in generated stems

## Dependencias

- T001: ToMatchSchema must exist

## Alcance

**In**:
1. Rewrite generateHierarchicalRootYAML for v2 format
2. Update buildHierarchicalStems to use ToMatchSchema
3. Update init_test.go tests
4. Verify generated stems pass rootline validate

**Out**: Non-hierarchical init (unchanged), migration (S002)

## Estado inicial esperado

- T001 completed: ToMatchSchema exists and works
- `cmd/rootline/init.go` exists with v1 generation logic

## Criterios de Aceptacion

- `go test ./cmd/rootline/ -run TestInit` passes — generated stems are v2 format
- Generated v2 .stem has `version: 2` and no `levels:` section
- Generated v2 .stem has fields with inline `match:` for per-level scoping
- `rootline validate` on generated stems passes
- `go test ./... -race` passes

## Fuente de verdad

- `cmd/rootline/init.go` — buildHierarchicalStems (~line 160), generateHierarchicalRootYAML (~line 200)
- `cmd/rootline/init_test.go` — Init command tests
