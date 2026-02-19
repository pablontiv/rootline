# S003: Describe Command

**Feature**: [F03 Validation and Schema](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: `rootline describe` muestra el schema efectivo de cualquier directorio con source tracing

## Antes / Despues

**Antes**: No hay forma de saber que schema aplica a un directorio sin leer manualmente todos los .stem ancestros. Los hooks hardcodean valores validos en vez de consultar el schema.

**Despues**: `rootline describe docs/prd/` muestra el schema efectivo completo en JSON, con `source` indicando cual .stem definio cada campo. `--field schema.Tipo.values` extrae valores especificos. Hooks y CI consultan `describe` en vez de hardcodear.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline describe docs/prd/` muestra schema con source tracing
- [ ] `--field` extrae valores especificos del schema
- [ ] Output es consumible por hooks de Claude Code y CI pipelines

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-describe-command.md) | Implementar cobra command `rootline describe` |

## Fuente de verdad

- `src/rootline/docs/research/I5-describe-contract.md` — contrato completo
