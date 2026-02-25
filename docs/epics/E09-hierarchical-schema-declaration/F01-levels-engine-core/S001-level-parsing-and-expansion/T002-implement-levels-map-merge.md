---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implement levels map merge in MergeStemFiles

**Story**: [S001 Level Parsing and Expansion](README.md)
**Contribuye a**: Levels map merge funciona correctamente in MergeStemFiles

[[blocks:T001-add-hierarchylevel-struct]]

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV3: `.stem` sin `levels:` zero regression
  - Verificar: `go test ./internal/rules/ -run TestMerge -v`

## Contexto

`MergeStemFiles` in `internal/rules/merge.go` merges a chain of StemEntries top-down (parent to child). Schema fields use map merge (child keys override parent), Validate uses array replacement, Derive/Aggregate use map merge. The new `Levels` field should follow **map merge** semantics: child `.stem` can add new levels or override existing ones by key, but non-overridden levels are preserved from parent.

Current merge logic in `merge.go` (lines 9-54) iterates over entries and merges each field type according to its YAML type. Adding Levels requires a map merge block similar to Schema.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: MergeStemFiles (extension)
    metodos:
      - nombre: MergeStemFiles
        input: entries []StemEntry
        output: "*StemFile"
dependencias_externas: []
tests:
  - TestMergeLevelsMap: parent levels + child levels produce map merge
  - TestMergeLevelsOverride: child overrides specific level schema
  - TestMergeLevelsNilChild: child without levels preserves parent levels
  - TestMergeLevelsNilParent: parent without levels, child adds levels
```

## Dependencias

- T001: HierarchyLevel struct must exist in StemFile

## Alcance

**In**:
1. Add levels map merge logic to `MergeStemFiles` in `internal/rules/merge.go` (~5 lines)
2. Add tests in `internal/rules/merge_test.go` for levels merge scenarios
3. Verify all existing merge tests still pass

**Out**: ExpandLevels (T003), nesting check (S002)

## Estado inicial esperado

- `HierarchyLevel` struct and `Levels` field exist in StemFile (from T001)
- `MergeStemFiles` handles Schema, Validate, Derive, Aggregate, Links, Structural

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestMergeLevels -v` passes
- `go test ./internal/rules/ -run TestMerge -v` passes unchanged (no regression)
- Parent `levels: {epic: ..., feature: ...}` + child `levels: {feature: ...override...}` merges correctly
- Child without levels preserves parent levels unchanged
- `go vet ./internal/rules/` reports no errors

## Fuente de verdad

- `internal/rules/merge.go` — MergeStemFiles function (lines 9-54)
- `internal/rules/merge_test.go` — existing merge tests
