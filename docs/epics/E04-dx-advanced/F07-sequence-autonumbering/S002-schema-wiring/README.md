---
tipo: historia
cliente: Platform Owner
---
# S002: Schema Wiring

**Feature**: [F07 Sequence Auto-numbering](../README.md)
**Capacidad**: Los `.stem` del arbol de planificacion (`docs/epics/`) definen campos `id` de tipo `sequence` en cada nivel jerarquico, habilitando `rootline describe --field schema.id.next` en cualquier directorio del arbol

## Antes / Despues

**Antes**: Solo `docs/epics/.stem` existe, sin campos `id`. Los subdirectorios de epics, features y stories no tienen `.stem`. No hay forma de obtener el proximo ID nativo.

**Despues**: Cada nivel del arbol tiene un `.stem` con el campo `id` correcto: epics/ define E+2digits, cada E*/F*/ define S+3digits, cada E*/F*/S*/ define T+3digits. `rootline describe docs/epics/ --field schema.id.next` retorna "E05". `rootline describe docs/epics/E04-dx-advanced/F07-sequence-autonumbering/S001-core-engine/ --field schema.id.next` retorna "T004".

## Criterios de Aceptacion (semanticos)

- [ ] `rootline describe docs/epics/ --field schema.id.next` retorna "E05"
- [ ] `rootline describe docs/epics/E04-dx-advanced/ --field schema.id.next` retorna "F09"
- [ ] `rootline describe docs/epics/E04-dx-advanced/F07-sequence-autonumbering/ --field schema.id.next` retorna "S003"
- [ ] `rootline describe docs/epics/E04-dx-advanced/F07-sequence-autonumbering/S001-core-engine/ --field schema.id.next` retorna "T004"

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-sequence-stems.md) | Crear/modificar .stem files en el arbol de planificacion con id sequence |

## Dependencias

- S001 completado (type: sequence implementado en el engine)

## Fuente de verdad

- `docs/epics/.stem` — .stem base a extender
- Directorios E04-dx-advanced/, F07-sequence-autonumbering/, S001-core-engine/ como ejemplos
