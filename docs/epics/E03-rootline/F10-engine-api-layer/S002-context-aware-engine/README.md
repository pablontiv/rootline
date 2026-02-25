---
estado: Specified
tipo: historia
---
# S002: Context-Aware Engine

**Feature**: [F10 Engine API Layer](../README.md)
**Capacidad**: Todas las operaciones core de rootline soportan cancelación y timeouts via context.Context

## Antes / Despues

**Antes**: Ninguna función en `internal/` acepta `context.Context`. Operaciones de scan/validate/derive no se pueden cancelar. Ctrl+C durante una validación masiva no tiene efecto limpio. El MCP server no podría manejar timeouts por request.

**Despues**: Todas las funciones públicas core aceptan `ctx context.Context` como primer parámetro. Operaciones largas (scan, validate --all) respetan cancelación. El MCP server puede imponer timeouts por request.

## Criterios de Aceptacion (semanticos)

- [ ] Funciones públicas de index, rules, derive, graph, query y extract aceptan context.Context
- [ ] Cancelar con Ctrl+C durante `rootline validate --all` en un repo grande termina limpiamente
- [ ] Todos los tests existentes siguen pasando (ctx = context.Background() en tests)

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-thread-context-core-interfaces.md) | Agregar context.Context a interfaces core de internal/ |
| [T002](T002-wire-cli-context.md) | Conectar cobra cmd.Context() a todas las llamadas internas |

## Fuente de verdad

- `internal/index/index.go` — Scan()
- `internal/rules/validate.go` — Validate()
- `internal/derive/derive.go` — DeriveAll(), AggregateAll()
- `internal/graph/graph.go` — Build()
- `internal/query/expr_eval.go` — ExecuteExpr()
