---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Add HierarchyLevel struct and levels parsing

**Story**: [S001 Level Parsing and Expansion](README.md)
**Contribuye a**: HierarchyLevel struct parsea correctamente desde YAML

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV3: `.stem` sin `levels:` zero regression
  - Verificar: `go test ./internal/rules/ -run TestMerge -v`

## Contexto

Rootline's `.stem` files define schemas for directories. The `StemFile` struct in `internal/rules/rules.go` has fields for Schema, Validate, Derive, Aggregate, Links, and Structural. We need to add a `Levels` field that maps level names to `HierarchyLevel` structs containing `match` (glob pattern), `children` (allowed child level names), `schema` (per-level schema fields), and `validate` (per-level validation rules).

The YAML format for levels:

```yaml
levels:
  epic:
    match: "E*"
    children: [feature]
    schema:
      id: { type: sequence, prefix: E, digits: 2 }
  feature:
    match: "F*"
    children: [story]
    schema:
      id: { type: sequence, prefix: F, digits: 2 }
```

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: HierarchyLevel
    metodos:
      - nombre: (struct fields)
        input: Match string, Children []string, Schema map[string]SchemaField, Validate []ValidationRule
        output: (struct)
dependencias_externas: []
tests:
  - TestHierarchyLevelParsing: YAML with levels unmarshals correctly
  - TestHierarchyLevelEmpty: StemFile without levels has nil Levels field
```

## Dependencias

- None — this is the first task

## Alcance

**In**:
1. Add `HierarchyLevel` struct to `internal/rules/rules.go` with fields: `Match string`, `Children []string`, `Schema map[string]SchemaField`, `Validate []ValidationRule`
2. Add `Levels map[string]*HierarchyLevel` field to `StemFile` struct
3. Add unit tests in `internal/rules/hierarchy_test.go` (new file) verifying YAML parsing of levels
4. Verify existing tests still pass

**Out**: ExpandLevels function (T003), merge logic (T002), CheckNesting (S002)

## Estado inicial esperado

- `internal/rules/rules.go` exists with StemFile struct (line 14-25)
- `SchemaField` and `ValidationRule` types already defined in same package
- No `Levels` field exists yet

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestHierarchyLevel -v` passes with struct parsing tests
- `go test ./internal/rules/ -run TestMerge -v` passes unchanged (no regression)
- `go vet ./internal/rules/` reports no errors
- StemFile with `levels:` YAML unmarshals with correct HierarchyLevel values
- StemFile without `levels:` YAML has `Levels == nil`

## Fuente de verdad

- `internal/rules/rules.go` — StemFile struct (lines 14-25), SchemaField, ValidationRule
- `internal/rules/rules_test.go` — existing tests (if any)
