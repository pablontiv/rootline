# S002: Query Command

**Feature**: [F04 Query and Presentation](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: `rootline query` busca y filtra documentos desde la linea de comandos

## Antes / Despues

**Antes**: Cada skill implementa su propia busqueda con grep, find, o Python regex. No hay interfaz unificada para consultar documentos. Agregar un nuevo tipo de query requiere modificar codigo en multiples skills.

**Despues**: `rootline query --where 'estado eq Pending' --from docs/epics/` retorna JSON rows. Multiples `--where` = implicit AND. `--field` extrae valores especificos. `--count` retorna numero. Interfaz unificada para todas las consultas.

## Criterios de Aceptacion (semanticos)

- [ ] Query con filtros retorna solo documentos que matchean
- [ ] Output JSON es parseable por scripts y AI assistants
- [ ] Multiples --where se combinan con AND

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-query-command.md) | Implementar cobra command `rootline query` |

## Fuente de verdad

- `src/rootline/docs/research/I1-query-operators.md` seccion 5 (CLI Flag Mapping)
