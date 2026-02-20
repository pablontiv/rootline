---
estado: Pending
tipo: feature
---
# F08: Edge Case Test Coverage

**Epic**: [E04](../README.md)
**Objetivo**: Los packages criticos de rootline tienen tests de edge cases que cubren escenarios de error, tipos de datos extremos, y comportamiento del CLI graph
**Beneficio**: Previene regresiones en escenarios no contemplados en los tests happy-path; en particular el comando `graph` no tiene ningun test a nivel CLI
**Milestone**: `go test ./... -race` pasa con coverage de edge cases en graph CLI, graph internal, merge, expr eval y extract

## Scope

**In**: Tests de edge cases en graph CLI (nuevo archivo), graph internal, merge/rules, expr eval, extract. Solo agregar tests, no modificar implementacion
**Out**: Tests de MCP server (stub), tests de comandos ya cubiertos (validate, query, fix, describe, tree, stats, new, init), refactors de implementacion

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-graph-cli-tests/) | Graph CLI Tests | `rootline graph` tiene cobertura de CLI completa |
| [S002](S002-internal-edge-cases/) | Internal Edge Cases | Packages internos tienen cobertura de edge cases |

## Dependencias

- Ninguna — independiente de F07

## Fuente de verdad

- `cmd/rootline/graph.go` — implementacion a testear
- `cmd/rootline/commands_test.go` — patron de tests CLI a seguir
- `internal/graph/graph_test.go` — tests internos a extender
- `internal/rules/merge_test.go` — tests de merge a extender
- `internal/query/expr_eval_test.go` — tests de expr a extender
- `internal/extract/extract_test.go` — tests de extract a extender
