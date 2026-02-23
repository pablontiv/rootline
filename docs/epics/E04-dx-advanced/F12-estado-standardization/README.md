---
tipo: feature
---
# F12: Estado System Standardization

**Epic**: [E04 DX Advanced](../README.md)
**Objetivo**: Sistema de estados estandarizado en ingles con semantica clara, hold manual, y aggregate correcto
**Beneficio**: Loop ejecuta solo tasks listos (Specified/In Progress), estados derivados (Blocked/On Hold) son automaticos, no hay valores <nil> en aggregate
**Milestone**: `rootline stats docs/epics/` muestra 0 valores <nil>, loop filtra por Specified/In Progress, hold field produce On Hold derivado

## Scope

**In**: Migracion de enum a ingles, reescritura de derive/aggregate expressions, campo hold, migracion de frontmatter, alineacion de codigo Go y skills
**Out**: Nuevos estados, cambios al derive engine, topological sort, aggregate engine changes

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-schema-data-migration/) | Schema & Data Migration | Enum, expresiones, y frontmatter migrados a ingles con hold y fallback correcto |
| [S002](S002-code-tooling-alignment/) | Code & Tooling Alignment | Codigo Go, tests, y skills alineados con nuevos valores en ingles |

## Orden de Ejecucion

| Story | Depende de | Razon |
|-------|-----------|-------|
| S001 | — | Schema y datos primero |
| S002 | S001 | e2e tests usan docs/epics/ como testdata |
