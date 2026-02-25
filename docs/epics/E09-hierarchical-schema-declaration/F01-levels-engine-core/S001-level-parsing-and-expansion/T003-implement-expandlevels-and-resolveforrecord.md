---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Implement ExpandLevels and ResolveForRecord

**Story**: [S001 Level Parsing and Expansion](README.md)
**Contribuye a**: ExpandLevels genera StemEntries virtuales correctos; ResolveForRecord produce effective schema identico a child .stem equivalentes

[[blocks:T002-implement-levels-map-merge]]

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV3: `.stem` sin `levels:` zero regression
  - Verificar: `go test ./internal/rules/ -run TestMerge -v`

## Contexto

This is the core function that makes `levels:` work. `ExpandLevels` takes a merged StemFile (with Levels populated) and a record's relative path, splits the path into components, matches each component against level `match` globs, and generates StemEntries with that level's schema/validate fields. These virtual entries are injected into the merge chain.

`ResolveForRecord` wraps the existing `WalkUp + MergeStemFiles` pattern:

```go
func ResolveForRecord(dir string, recordPath string) (*StemFile, error) {
    entries, err := WalkUp(dir)
    if err != nil { return nil, err }
    merged := MergeStemFiles(entries)
    if merged.Levels != nil {
        virtualEntries := ExpandLevels(merged, recordPath)
        allEntries := append(entries, virtualEntries...)
        merged = MergeStemFiles(allEntries)
    }
    return merged, nil
}
```

For a record at `E01/F02/S001/T001.md`, ExpandLevels matches:
- `E01` against `epic` level (match: "E*") — generates StemEntry with epic schema
- `F02` against `feature` level (match: "F*") — generates StemEntry with feature schema
- `S001` against `story` level (match: "S*") — generates StemEntry with story schema
- `T001.md` against `task` level (match: "T*") — generates StemEntry with task schema

Virtual entries are generated in path order (shallowest first) so merge applies them progressively.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: ExpandLevels
    metodos:
      - nombre: ExpandLevels
        input: "stem *StemFile, recordRelPath string"
        output: "[]StemEntry"
  - nombre: ResolveForRecord
    metodos:
      - nombre: ResolveForRecord
        input: "dir string, recordPath string"
        output: "*StemFile, error"
dependencias_externas: []
tests:
  - TestExpandLevels: generates correct virtual entries by depth for E/F/S/T path
  - TestExpandLevelsNoMatch: path component not matching any level generates no entry
  - TestExpandLevelsPartialMatch: only some components match levels
  - TestExpandLevelsWithRealChild: real child .stem + virtual level — real wins via merge order
  - TestResolveForRecord: produces same effective schema as equivalent child .stem files
  - TestResolveForRecordNoLevels: without levels, identical to WalkUp+Merge
```

## Dependencias

- T002: Levels map merge must work in MergeStemFiles

## Alcance

**In**:
1. Create `internal/rules/hierarchy.go` with `ExpandLevels` and `ResolveForRecord`
2. `ExpandLevels`: split path by `/`, match each component against all levels using `filepath.Match`, generate StemEntry per match
3. `ResolveForRecord`: WalkUp + Merge + conditional ExpandLevels + re-Merge
4. Add comprehensive tests in `internal/rules/hierarchy_test.go`

**Out**: CheckNesting (S002/T001), caller migration (F02)

## Estado inicial esperado

- `HierarchyLevel` struct exists with Match, Children, Schema, Validate (T001)
- `MergeStemFiles` handles Levels map merge (T002)
- `WalkUp` function exists in `internal/rules/` for stem discovery
- `StemEntry` type exists for the merge chain

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestExpandLevels -v` passes all expansion scenarios
- `go test ./internal/rules/ -run TestResolveForRecord -v` passes
- A test fixture with `levels:` produces identical effective schema to equivalent child `.stem` chain
- Without `levels:`, ResolveForRecord behaves identically to WalkUp+MergeStemFiles
- `go test ./... -race` passes (no regression)

## Fuente de verdad

- `internal/rules/rules.go` — StemFile, StemEntry, HierarchyLevel
- `internal/rules/merge.go` — MergeStemFiles, WalkUp
- `internal/rules/hierarchy.go` — (new) ExpandLevels, ResolveForRecord
