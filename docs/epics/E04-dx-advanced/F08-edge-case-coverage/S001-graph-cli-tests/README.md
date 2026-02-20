---
estado: Specified
tipo: historia
cliente: Platform Owner
---
# S001: Graph CLI Tests

**Feature**: [F08 Edge Case Coverage](../README.md)
**Capacidad**: El comando `rootline graph` tiene tests de CLI que cubren JSON output, --check mode, --format dot/mermaid, y error handling

## Antes / Despues

**Antes**: `rootline graph` es el unico comando sin tests a nivel CLI. Solo existe `internal/graph/graph_test.go` que testea el package interno. Los paths de codigo de --check, renderDOT, renderMermaid, y error handling no tienen cobertura.

**Despues**: `cmd/rootline/graph_test.go` cubre los 8 escenarios principales del comando graph. `go test ./cmd/rootline/ -run TestGraph -v` pasa.

## Criterios de Aceptacion (semanticos)

- [ ] `go test ./cmd/rootline/ -run TestGraph -v` pasa con 8 test cases
- [ ] Coverage incluye: JSON empty, JSON with links, --check clean, --check cycle, --check broken, --format dot, --format mermaid, invalid format

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-graph-cli-tests.md) | Crear cmd/rootline/graph_test.go con 8 tests |

## Fuente de verdad

- `cmd/rootline/graph.go` — implementacion a testear
- `cmd/rootline/commands_test.go` — patron runCmd/setupTestDir a reusar
- `internal/extract/links.go` — formato [[target]] para crear fixtures con links
