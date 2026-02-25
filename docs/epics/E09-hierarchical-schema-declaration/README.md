---
estado: Pending
tipo: feature
---
# E09: Hierarchical Schema Declaration

**Metrica de exito**: Un solo `.stem` file con `levels:` produce el mismo effective schema que N child `.stem` files equivalentes, y detecta nesting violations
**Timeline**: 2026-Q1 — en curso

## Intencion

Rootline requiere un `.stem` file por cada nivel de la jerarquia (epic, feature, story, task) para declarar schemas diferenciados. Esto genera proliferacion de archivos repetitivos (el proyecto rootline tiene ~70 child `.stem` files, homeserver tiene uno flat sin diferenciacion por nivel). Este Epic introduce `levels:` como azucar sintactico en `.stem` files: una seccion que declara schemas per-level en un solo archivo, expandiendose internamente a StemEntries virtuales durante el merge. Ademas, habilita validacion de nesting (impedir que un task exista directo bajo un epic).

## Postcondiciones

| ID | Condicion | Features |
|----|-----------|----------|
| P1 | `.stem` con `levels:` produce el mismo effective schema que child `.stem` equivalentes | F01, F02 |
| P2 | Nesting violations (task bajo epic) son detectadas por `rootline validate` | F01 |
| P3 | `.stem` files existentes sin `levels:` funcionan sin cambio | F01, F02 |
| P4 | homeserver/epics valida correctamente con un solo `.stem` con `levels:` | F03 |
| P5 | rootline/docs/epics consolida ~70 child `.stem` en uno solo con `levels:` | F03 |

## Invariantes

- INV1: Todos los tests existentes pasan sin modificacion
- INV2: Coverage se mantiene >= 85%

## Out of Scope

- Expresiones `expr` en `match:` (solo glob patterns)
- Refactoring del derive/aggregate engine
- Cambios al MCP protocol o JSON-RPC layer
- Migracion automatica de child `.stem` a `levels:` (se hace manual)

## Features

| ID | Nombre | Descripcion |
|----|--------|-------------|
| [F01](F01-levels-engine-core/) | Levels Engine Core | Parsing, expansion a StemEntries virtuales, y validacion de nesting |
| [F02](F02-caller-migration-and-tooling/) | Caller Migration and Tooling | Migrar callers a ResolveForRecord, stemhealth checks, infer updates |
| [F03](F03-schema-migration/) | Schema Migration | Migrar homeserver y rootline docs/epics a `levels:` |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | Foundation: implementa el engine de levels |
| F02 | F01 | Callers necesitan ResolveForRecord que depende de ExpandLevels |
| F03 | F02 | Migracion requiere que todo el pipeline soporte levels |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-02-25 | `levels:` como azucar sintactico (expand to virtual StemEntries) | Minimiza cambios: validate, derive, query no cambian |
| 2026-02-25 | Coexistencia: child `.stem` real override virtual de levels | Backward compatible, merge natural |
| 2026-02-25 | Nesting check como paso de validacion, no en merge | Separacion de concerns: merge resuelve schema, validate chequea estructura |

## Gaps Activos

- Ninguno identificado
