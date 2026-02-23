---
tipo: feature
---
# E07: Schema Evolution

**Estado**: Completado
**Metrica de exito**: `rootline migrate --dry-run` detecta breaking changes en .stem; `rootline migrate --rename` actualiza campo en N archivos atomicamente
**Timeline**: 2026-Q1

## Intencion

Cuando un .stem cambia, documentos existentes pueden quedar invalidos. Hoy `rootline fix --all` corrige valores (typos, enums, campos faltantes), pero no maneja cambios de schema: renombrar campos, detectar breaking changes, ni mantener historial de migraciones. Esta epic agrega `rootline migrate` como complemento de `fix` — orientado a cambios de schema, no de datos.

## Features

| Feature | Descripcion |
|---------|-------------|
| [F01 Migration Tooling](F01-migration-tooling/) | Deteccion de breaking changes y operaciones bulk de migracion |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | Sin dependencias externas (usa rules + fix patterns existentes) |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-02-22 | git-based comparison como primary, --from como fallback | .stem no tiene versionado propio; git es la fuente de verdad |
| 2026-02-22 | Migration log como JSON Lines append-only | Simple, git-friendly, parseable |

## Gaps Activos

- Multi-level .stem diffs (cambio en parent .stem afecta todos los children)
- Migraciones interactivas (seleccionar que cambios aplicar)
