---
tipo: historia
cliente: Platform Owner
---
# S001: Schema & Data Migration

**Feature**: [F12 Estado System Standardization](../README.md)
**Capacidad**: Enum de estados en ingles con hold field, derive/aggregate expressions correctas, y frontmatter migrado

## Antes / Despues

**Antes**: 6 estados mezclando espanol e ingles (Pending, In Progress, Specified, Completado, Diferida, Bloqueada). Aggregate fallback retorna `<nil>` en index files sin estado en frontmatter. No hay mecanismo para bloqueo manual por usuario. Loop filtra por lista inflada de 6 valores.

**Despues**: 6 estados en ingles con semantica clara (Pending, Specified, In Progress, Completed, Blocked, On Hold). Aggregate fallback retorna `"Pending"`. Campo `hold: string` en .stem; derive produce `On Hold` cuando hold existe. Frontmatter de 141 archivos migrado a valores en ingles.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline validate --all docs/epics/` retorna 0 invalid
- [ ] `rootline query docs/epics/ --where "estado == nil"` retorna 0 resultados
- [ ] `rootline query docs/epics/ --where "estado == 'Completed'"` retorna 137+ records
- [ ] `rootline query docs/epics/ --where "estado == 'Specified'"` retorna 4 tasks (E03/F05)

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-update-stem-schema.md) | Update .stem enum, derive, aggregate, and hold field |
| [T002](T002-migrate-frontmatter-values.md) | Bulk migrate frontmatter values across docs/epics |
| [T003](T003-validate-migration.md) | Validate migration with rootline CLI |

## Fuente de verdad

- `docs/epics/.stem` (enum, derive, aggregate, hold)
- `docs/epics/**/*.md` (frontmatter estado values)
