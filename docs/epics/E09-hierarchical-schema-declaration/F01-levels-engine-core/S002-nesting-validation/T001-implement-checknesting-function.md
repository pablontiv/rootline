---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implement CheckNesting function

**Story**: [S002 Nesting Validation](README.md)
**Contribuye a**: CheckNesting valida cadena E->F->S->T y detecta nesting violations

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV3: `.stem` sin `levels:` zero regression
  - Verificar: CheckNesting with nil levels returns no errors

## Contexto

`CheckNesting` validates that each path component's level is an allowed child of its parent's level, using the `children` field in HierarchyLevel. This is a **validation step**, not a merge step — it runs after ResolveForRecord to check structural correctness.

For path `E01/F02/S001/T001.md` with levels `{epic: {children: [feature]}, feature: {children: [story]}, story: {children: [task]}, task: {children: []}}`:
- `E01` (epic) → root level → OK (no parent constraint)
- `F02` (feature) → parent is epic → `epic.children` includes "feature" → OK
- `S001` (story) → parent is feature → `feature.children` includes "story" → OK
- `T001.md` (task) → parent is story → `story.children` includes "task" → OK

For `E01/T001.md` (task directly under epic):
- `E01` (epic) → OK
- `T001.md` (task) → parent is epic → `epic.children = [feature]` → ERROR: "task" not in children

Leaf constraint: `task.children = []` means no subdirectories should exist under a task.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: CheckNesting
    metodos:
      - nombre: CheckNesting
        input: "levels map[string]*HierarchyLevel, recordRelPath string"
        output: "[]ValidationError"
dependencias_externas: []
tests:
  - TestCheckNesting: valid chain E->F->S->T returns no errors
  - TestCheckNestingInvalid: task under epic returns error
  - TestCheckNestingLeaf: subdir under leaf (children:[]) returns error
  - TestCheckNestingNoLevels: nil levels returns no errors
  - TestCheckNestingUnknownComponent: component not matching any level is skipped
```

## Dependencias

- F01/S001/T001: HierarchyLevel struct must exist

## Alcance

**In**:
1. Add `CheckNesting` function to `internal/rules/hierarchy.go`
2. For each path component, resolve its level name by matching against level patterns
3. Check that child level name is in parent level's `children` list
4. Return `[]ValidationError` for violations
5. Add tests in `internal/rules/hierarchy_test.go`

**Out**: Integration into validate/fix pipelines (T002), caller migration (F02)

## Estado inicial esperado

- `internal/rules/hierarchy.go` exists with ExpandLevels and ResolveForRecord (from S001)
- HierarchyLevel struct has Children field
- ValidationError type exists in the rules package

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestCheckNesting -v` passes all scenarios
- Valid E->F->S->T path returns empty error slice
- Task under epic returns error with descriptive message
- Nil levels returns empty errors (skip)
- `go test ./... -race` passes

## Fuente de verdad

- `internal/rules/hierarchy.go` — ExpandLevels, CheckNesting
- `internal/rules/rules.go` — HierarchyLevel, ValidationError types
