# F04: Query and Presentation

**Epic**: [E03](../README.md)
**Objetivo**: Rootline responde queries estructuradas y muestra vistas jerarquicas
**Beneficio**: Reemplaza 18 patrones de query de consumidores con un unico modelo declarativo
**Milestone**: `rootline query --where 'estado eq Pending'` retorna JSON rows correctos

## Scope

**In**: Query engine (5 operators + and + count + limit), field shortcuts, query command, tree command, stats command
**Out**: order_by, or operator, derived state computation, graph queries

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Query Engine](S001-query-engine/) | Motor de queries con 5 operadores + and + count + limit |
| S002 | [Query Command](S002-query-command/) | `rootline query` CLI command |
| S003 | [Tree and Stats](S003-tree-stats/) | `rootline tree` y `rootline stats` CLI commands |

## Dependencias

- F02 completado (core engine: extraction + scanner produce Records)

## Fuente de verdad

- `src/rootline/docs/research/I1-query-operators.md` — operadores y JSON contract
- `src/rootline/docs/intent/v0-rootline.md` — seccion 3 (Commands)
