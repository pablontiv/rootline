---
estado: Completado
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Crear .stem files con id sequence en el arbol de planificacion

**Story**: [S002 Schema Wiring](README.md)

## Contexto

El engine de sequence ya funciona (S001 completado), pero ninguno de los directorios del arbol de planificacion tiene configurado el campo `id`. Se necesita agregar la definicion de sequence al `.stem` de cada nivel jerarquico. El merge top-down de rootline permite que un `.stem` padre defina el campo `id` y los hijos lo hereden, pero como cada nivel tiene prefix distinto, cada nivel necesita su propio `.stem` con el campo `id` correcto.

## Alcance

**In**:
1. Modificar `docs/epics/.stem` — agregar `id: {type: sequence, prefix: E, digits: 2}` bajo `schema:`
2. Crear `.stem` pattern para nivel Feature: cada `docs/epics/E*/` debe tener `.stem` con `id: {type: sequence, prefix: F, digits: 2}`. Crear uno de referencia en `docs/epics/E04-dx-advanced/.stem` (los otros se crean on-demand)
3. Crear `.stem` pattern para nivel Story: `docs/epics/E04-dx-advanced/F07-sequence-autonumbering/.stem` con `id: {type: sequence, prefix: S, digits: 3}`
4. Crear `.stem` pattern para nivel Task: `docs/epics/E04-dx-advanced/F07-sequence-autonumbering/S001-core-engine/.stem` con `id: {type: sequence, prefix: T, digits: 3}`

**Out**: Cambios al engine, agregar .stem a todos los directorios existentes (solo los 4 niveles de referencia)

## Estado inicial esperado

- S001 completado: engine acepta type: sequence y computa next
- `docs/epics/.stem` existe con schema de estado y tipo

## Criterios de Aceptacion

- `rootline describe docs/epics/ --field schema.id.next` retorna "E05" (o el siguiente segun epics existentes)
- `rootline describe docs/epics/E04-dx-advanced/ --field schema.id.next` retorna el siguiente Feature
- `rootline describe docs/epics/E04-dx-advanced/F07-sequence-autonumbering/ --field schema.id.next` retorna el siguiente Story
- `rootline describe docs/epics/E04-dx-advanced/F07-sequence-autonumbering/S001-core-engine/ --field schema.id.next` retorna "T004"
- `rootline validate --all docs/epics/` pasa (no regresiones en validacion existente)

## Fuente de verdad

- `docs/epics/.stem` — archivo a modificar
- `docs/epics/E04-dx-advanced/.stem` — archivo a crear
- `docs/epics/E04-dx-advanced/F07-sequence-autonumbering/.stem` — archivo a crear
- `docs/epics/E04-dx-advanced/F07-sequence-autonumbering/S001-core-engine/.stem` — archivo a crear
