# S001: Query Engine

**Feature**: [F04 Query and Presentation](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Motor de queries declarativo filtra Records con 5 operadores derivados de 18 consumidores reales

## Antes / Despues

**Antes**: 18 consumidores usan grep/regex/Python splits para filtrar documentos. 4 patrones regex distintos para el mismo campo `estado:`. Failures silenciosos cuando el formato cambia. Cada skill reimplementa filtering independientemente.

**Despues**: Query engine declarativo con 5 operadores (eq, ne, in, contains, exists) + and logico + count/limit. JSON contract versionado. Shortcuts para campos comunes. 100% de los 18 patrones de consumidores cubiertos.

## Criterios de Aceptacion (semanticos)

- [ ] Query con `eq` filtra correctamente por valor exacto
- [ ] Query con `in` filtra por multiples valores
- [ ] Query con `contains` busca en body del documento
- [ ] `count` retorna numero en vez de rows

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-query-engine-operators.md) | Implementar 5 operadores + and + count + limit |
| [T002](T002-field-shortcuts.md) | Resolver shortcuts de campos comunes |

## Fuente de verdad

- `src/rootline/docs/research/I1-query-operators.md` — spec completa de operadores
