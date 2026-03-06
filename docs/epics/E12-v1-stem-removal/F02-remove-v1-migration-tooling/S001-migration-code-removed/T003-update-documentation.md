---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T003: Actualizar documentacion

**Story**: [S001 Codigo de migracion v1 eliminado](README.md)
**Contribuye a**: Documentacion no referencia v1 como formato soportado

[[blocks:T002-remove-cli-flags]]

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`

## Contexto

Con el codigo y CLI de migracion v1 eliminados (T001, T002), la documentacion debe reflejar que v1 ya no es soportado. Tres archivos tienen referencias a v1: `docs/levels.md` (seccion deprecated y migracion), `docs/migrate.md` (flags --from-levels y --to-v2), y `docs/research/inference-engine-architecture.md` (linea 22 dice que flat mode genera v1, pero ya genera v2).

## Alcance

**In**:
1. `docs/levels.md`: quitar seccion deprecated y referencias a migracion v1
2. `docs/migrate.md`: quitar documentacion de `--from-levels` y `--to-v2`
3. `docs/research/inference-engine-architecture.md`: corregir linea 22 (flat mode genera v2, no v1)

**Out**: No crear nueva documentacion. No tocar CLAUDE.md.

## Estado inicial esperado

- T002 completado: flags CLI eliminados
- Documentacion todavia referencia v1

## Criterios de Aceptacion

- `grep -n "from-levels\|--to-v2" docs/migrate.md` retorna 0 resultados
- `grep -n "version: 1.*flat" docs/research/inference-engine-architecture.md` retorna 0 resultados
- `grep -in "deprecated.*v1\|v1.*deprecated" docs/levels.md` retorna 0 resultados

## Fuente de verdad

- `docs/levels.md`
- `docs/migrate.md`
- `docs/research/inference-engine-architecture.md`
