---
tipo: historia
---
# S001: Fix Priority Conflicts

**Feature**: [F08 Proposal Engine Fixes](../README.md)
**Cliente**: Platform Owner

**Capacidad**: Los detectores del proposal engine respetan una jerarquia de prioridad, eliminando proposals conflictivos para el mismo path/field. `rootline fix --all` aplica proposals sin corromper datos.

## Antes / Despues

**Antes**: Detectores generan proposals independientemente. `migrate_value` y `correct_value` compiten por el mismo archivo — el ultimo write gana, perdiendo wiki-links. `add_field` pisa valores de `extract_body` e `infer_from_children`. `correct_value` revierte cambios de `extend_enum`. Desde repo root, `fix --all` toma un stem aleatorio del map y puede perder el schema con enums. `tree` no usa ScopeResolver e incluye archivos fuera de scope.

**Despues**: Prioridad definida: `migrate_value` > `correct_value`, `extract_body`/`infer_from_children` > `add_field`, `extend_enum` invalida `correct_value` stale. Stem selection determinista por riqueza de schema. `tree` usa ScopeResolver.

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-fix-stem-selection.md) | Seleccion determinista de stem por riqueza de schema |
| [T002](T002-dedup-migrate-vs-correct.md) | migrate_value tiene prioridad sobre correct_value |
| [T003](T003-skip-correct-after-extend.md) | Skip correct_value cuando extend_enum hizo el valor valido |
| [T004](T004-dedup-infer-vs-addfield.md) | extract_body/infer_from_children tienen prioridad sobre add_field |
| [T005](T005-tree-scope-resolver.md) | Agregar ScopeResolver a tree command |

## Fuente de verdad

- `internal/proposal/proposal.go`
- `cmd/rootline/fix.go`
- `cmd/rootline/tree.go`
