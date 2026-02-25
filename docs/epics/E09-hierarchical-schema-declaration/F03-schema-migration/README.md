---
estado: Pending
tipo: feature
---
# F03: Schema Migration

**Epic**: [E09](../README.md)
**Satisface**: P4, P5
**Objetivo**: Los proyectos homeserver y rootline usan `levels:` en sus `.stem` files raiz, eliminando child `.stem` repetitivos
**Beneficio**: Dogfooding del feature levels con los dos proyectos principales, validando que la funcionalidad es correcta y practica
**Milestone**: `rootline validate --all` pasa en ambos proyectos con un solo `.stem` con `levels:` por jerarquia

## Scope

**In**: Reescribir homeserver `docs/epics/.stem` con levels; consolidar ~70 child `.stem` de rootline en root `.stem` con levels
**Out**: Automatizacion de migracion (se hace manual), cambios al engine

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-migrate-homeserver-stem-to-levels/) | Migrate homeserver .stem to levels | homeserver/docs/epics usa un solo `.stem` con schema diferenciado por nivel |
| [S002](S002-migrate-rootline-docs-epics-stem-to-levels/) | Migrate rootline docs/epics .stem to levels | rootline/docs/epics consolida ~70 child `.stem` en uno solo con overrides donde necesario |

## Invariantes

- INV1 (heredado): Todos los tests existentes pasan sin modificacion
- INV5: Todos los documentos existentes siguen validando correctamente post-migracion

## Dependencias

- F02: El pipeline completo debe soportar levels antes de migrar archivos reales

## Fuente de verdad

- `/opt/homeserver/automation/docs/epics/.stem` — homeserver stem actual
- `/opt/rootline/docs/epics/.stem` — rootline root stem
- `/opt/rootline/docs/epics/E03-rootline/.stem` — ejemplo de child stem a consolidar
