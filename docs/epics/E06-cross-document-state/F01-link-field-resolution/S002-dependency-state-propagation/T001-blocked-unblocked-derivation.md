---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Configurar derive expression para blocked/unblocked en .stem

**Story**: [S002 Dependency State Propagation](README.md)

[[blocks:T002-derive-env-linked-values]]

## Contexto

Con S001 completado, el derive engine inyecta `blocked_by` como slice de valores del campo referenciado. Esta task configura el .stem con la expression correcta y migra los documents existentes que tienen `estado: Pending (blocked by X)` al formato correcto.

## Alcance

**In**:
1. Agregar `links.blocks.field: blocked_by` al .stem de docs/epics/
2. Agregar derive expression para estado que usa blocked_by
3. Migrar tasks con `estado: Pending (blocked by X)` → `estado: Bloqueada` + `[[blocks:TXXX]]` en body
4. Verificar que `rootline query --where "estado == 'Pending'"` retorna solo tasks accionables

**Out**: Cambios al aggregate expression existente, UI changes, nuevos link types

## Estado inicial esperado

- S001 completado (linked values en derive env)
- docs/epics/.stem con links.blocks ya definido

## Criterios de Aceptacion

- .stem tiene `links.blocks.field: blocked_by`
- .stem tiene derive expression: `blocked_by != nil && all(blocked_by, ...) ? 'Pending' : ...`
- Tasks previamente con texto libre en estado ahora tienen enum valido + wiki-links
- `rootline validate --all docs/epics/` pasa sin errores de enum en estado

## Fuente de verdad

- `docs/epics/.stem`
- Tasks con `estado: Pending (blocked by X)` (a migrar)
