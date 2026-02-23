---
estado: Pending
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Bulk migrate frontmatter values across docs/epics

**Story**: [S001 Schema & Data Migration](README.md)

[[blocks:T001-update-stem-schema]]

## Contexto

Con T001 completado, el .stem tiene el nuevo enum en ingles. Pero los 141 archivos de frontmatter (tasks, no READMEs) todavia usan valores viejos: `Completado` (137 files) y `Pending` (4 files). Los valores deben migrar a `Completed` y `Specified` respectivamente.

## Alcance

**In**:
1. Reemplazar `estado: Completado` → `estado: Completed` en todos los archivos bajo `docs/epics/`
2. Reemplazar `estado: Pending` → `estado: Specified` en los 4 tasks pendientes (todos tienen spec completa)
3. Verificar que no queden valores en espanol (`Completado`, `Bloqueada`, `Diferida`)

**Out**: READMEs (ya no tienen estado), cambios a codigo, validacion completa (T003)

## Estado inicial esperado

- T001 completado (enum en ingles en .stem)
- 137 archivos con `estado: Completado`
- 4 archivos con `estado: Pending`
- 0 archivos con `estado: Bloqueada` o `estado: Diferida`

## Criterios de Aceptacion

- `grep -r "estado: Completado" docs/epics/ | wc -l` retorna 0
- `grep -r "estado: Pending" docs/epics/ | wc -l` retorna 0
- `grep -r "estado: Bloqueada" docs/epics/ | wc -l` retorna 0
- `grep -r "estado: Completed" docs/epics/ | wc -l` retorna 137
- `grep -r "estado: Specified" docs/epics/ | wc -l` retorna 4+

## Fuente de verdad

- `docs/epics/**/*.md` (frontmatter)
