---
estado: Pending
---
# S001: Migrate homeserver .stem to levels

**Feature**: [F03 Schema Migration](../README.md)
**Capacidad**: homeserver/docs/epics usa un solo `.stem` con schema diferenciado por nivel
**Cubre**: P4 del Epic — homeserver valida correctamente con levels

## Antes / Despues

**Antes**: `/opt/homeserver/automation/docs/epics/.stem` es flat — un solo schema para 293 documentos en 4 niveles. Features sin `tipo` requerido, stories sin diferenciacion, tasks sin `ejecutable_en` obligatorio.

**Despues**: El mismo `.stem` declara `levels:` con schemas diferenciados: epics tienen solo `id`, features agregan `tipo`, tasks agregan `ejecutable_en` como required. `rootline validate --all` detecta documentos que no cumplen su schema de nivel.

## Criterios de Aceptacion (semanticos)

- [ ] `.stem` reescrito con `levels:` section
- [ ] `rootline validate --all /opt/homeserver/automation/docs/epics/` pasa sin errores de schema
- [ ] Documentos en nivel incorrecto son detectados si existen

## Invariantes

- INV5: Todos los documentos existentes siguen validando correctamente
  - Verificar: `rootline validate --all /opt/homeserver/automation/docs/epics/`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-rewrite-homeserver-stem-with-levels.md) | Rewrite homeserver epics .stem with levels and validate |

## Fuente de verdad

- `/opt/homeserver/automation/docs/epics/.stem` — current flat stem
