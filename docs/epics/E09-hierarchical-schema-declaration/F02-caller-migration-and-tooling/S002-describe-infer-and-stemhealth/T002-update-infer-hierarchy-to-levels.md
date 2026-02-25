---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Update infer --hierarchy to generate levels format

**Story**: [S002 Describe, Infer and Stemhealth](README.md)
**Contribuye a**: infer --hierarchy genera formato levels

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`

## Contexto

`internal/infer/hierarchy.go` implements `InferHierarchy` which analyzes directory naming patterns (E##, F##, S###, T###) and generates per-level `.stem` content. Currently it outputs separate child `.stem` files or a flat schema. With `levels:` support, it should generate a single `.stem` with `levels:` section that declares per-level schemas.

The existing detection logic (pattern matching, field distribution by level) remains useful — it just needs to output the result as a `levels:` YAML block instead of separate files.

## Dependencias

- F01/S001/T001: HierarchyLevel struct for output format

## Alcance

**In**:
1. Update `internal/infer/hierarchy.go` to generate `levels:` format output
2. Each detected level gets a HierarchyLevel entry with match pattern, children, and level-specific schema
3. Update tests in `internal/infer/hierarchy_test.go`
4. Ensure `rootline infer --hierarchy <dir>` outputs a single `.stem` with levels

**Out**: migrate --split (separate feature), actual .stem migration (F03)

## Estado inicial esperado

- `internal/infer/hierarchy.go` exists with InferHierarchy function
- It already detects directory naming patterns and distributes fields by level
- HierarchyLevel struct exists

## Criterios de Aceptacion

- `go test ./internal/infer/ -run TestHierarchy -v` passes
- `rootline infer --hierarchy <dir>` outputs `.stem` YAML with `levels:` section
- Generated levels have correct match patterns (E*, F*, S*, T*)
- Generated levels have correct children chains
- Per-level schema fields are distributed correctly
- `go test ./... -race` passes

## Fuente de verdad

- `internal/infer/hierarchy.go` — InferHierarchy function
- `internal/infer/hierarchy_test.go` — existing hierarchy inference tests
