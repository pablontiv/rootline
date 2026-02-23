---
estado: Pending
tipo: historia
cliente: Platform Owner
---
# S001: Change Detection

**Feature**: [F01 Migration Tooling](../README.md)
**Capacidad**: rootline migrate --dry-run compara .stem actual vs anterior y reporta breaking changes con clasificacion de severidad

## Antes / Despues

**Antes**: Cambias un .stem (renombras campo, eliminas enum value), ejecutas rootline validate, y ves N errores sin contexto de que cambio. No hay forma de saber si un cambio es breaking antes de aplicarlo.

**Despues**: `rootline migrate --dry-run` compara el .stem actual contra la version anterior (via git o --from) y reporta: que cambio, si es breaking, cuantos archivos afecta. El developer sabe exactamente el impacto antes de commitear.

## Criterios de Aceptacion (semanticos)

- [ ] rootline migrate --dry-run muestra diff de .stem
- [ ] Breaking changes clasificados: field removed, enum removed, type changed, required tightened
- [ ] Non-breaking cambios identificados: field added, enum added, required relaxed
- [ ] Conteo de archivos afectados por cada breaking change

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-migrate-dry-run.md) | Implementar rootline migrate --dry-run |
| [T002](T002-breaking-change-classifier.md) | Clasificador de breaking changes |

## Fuente de verdad

- `internal/rules/rules.go` (StemFile struct)
- git (fuente de version anterior de .stem)
