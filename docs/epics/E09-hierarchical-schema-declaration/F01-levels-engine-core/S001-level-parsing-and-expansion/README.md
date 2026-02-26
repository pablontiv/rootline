---
estado: Pending
---
# S001: Level Parsing and Expansion

**Feature**: [F01 Levels Engine Core](../README.md)
**Capacidad**: El engine parsea `levels:` de `.stem` files y genera StemEntries virtuales que producen el effective schema correcto per-record
**Cubre**: Milestone de F01 — expansion correcta de levels a effective schema

## Antes / Despues

**Antes**: Para declarar schemas diferenciados por nivel (epic, feature, story, task), se requiere un `.stem` file fisico en cada directorio intermedio. Un proyecto con 4 niveles necesita al menos 4 `.stem` files con contenido repetitivo.

**Despues**: Un solo `.stem` file con seccion `levels:` declara schemas per-level. El engine expande internamente cada level a un StemEntry virtual, produciendo el mismo effective schema que N child `.stem` files. `MergeStemFiles` no sabe que algunos entries son virtuales.

## Criterios de Aceptacion (semanticos)

- [ ] `HierarchyLevel` struct parsea correctamente desde YAML con match, children, schema, validate
- [ ] `ExpandLevels` genera StemEntries virtuales correctos dado un path relativo
- [ ] `ResolveForRecord` produce el mismo effective schema que child `.stem` equivalentes
- [ ] Levels map merge funciona correctamente en `MergeStemFiles` (child override parent)
- [ ] `.stem` files sin `levels:` producen el mismo resultado que antes

## Invariantes

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV2: Coverage >= 85%
  - Verificar: `go test ./... -coverprofile=c.out && go tool cover -func=c.out | tail -1`
- INV3: `.stem` sin `levels:` zero regression
  - Verificar: `go test ./internal/rules/ -run TestMerge -v`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-hierarchylevel-struct.md) | Add HierarchyLevel struct and levels parsing to StemFile |
| [T002](T002-implement-levels-map-merge.md) | Implement levels map merge in MergeStemFiles |
| [T003](T003-implement-expandlevels-and-resolveforrecord.md) | Implement ExpandLevels and ResolveForRecord functions |
| [T004](T004-fix-merge-severity-default.md) | Fix mergeFieldSeverity empty severity default |

## Fuente de verdad

- `internal/rules/rules.go` — StemFile, SchemaField, ValidationRule structs
- `internal/rules/merge.go` — MergeStemFiles function
- `internal/rules/merge_test.go` — existing merge tests
