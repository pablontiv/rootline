---
estado: Specified
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Validate migration with rootline CLI

**Story**: [S001 Schema & Data Migration](README.md)

[[blocks:T002-migrate-frontmatter-values]]

## Contexto

Con T001 y T002 completados, el .stem tiene enum en ingles y los frontmatter estan migrados. Esta task valida que todo es consistente usando los comandos de rootline de menos a mas agresivos.

## Alcance

**In**:
1. `rootline validate --all docs/epics/` — 0 invalid
2. `rootline query docs/epics/ --where "estado == nil"` — 0 resultados
3. `rootline query docs/epics/ --where "estado == 'Completed'" --output table` — 137+ records
4. `rootline query docs/epics/ --where "estado == 'Specified'" --output table` — 4+ records
5. `rootline doctor docs/epics/` — no new failures
6. `rootline stats docs/epics/ --output table` — 0 `<nil>` values

**Out**: Cambios a codigo, cambios a skills

## Estado inicial esperado

- T001 y T002 completados
- .stem con enum en ingles, derive con hold, aggregate con fallback "Pending"
- Frontmatter migrado a Completed/Specified

## Criterios de Aceptacion

- `rootline validate --all docs/epics/` → 0 invalid files
- `rootline query docs/epics/ --where "estado == nil"` → 0 rows
- `rootline stats docs/epics/ --output table` → 0 `<nil>` en By Estado
- `rootline doctor docs/epics/` → 0 new failures (1 preexisting type-consistency is OK)

## Fuente de verdad

- `docs/epics/.stem`
- `docs/epics/**/*.md`
