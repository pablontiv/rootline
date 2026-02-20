---
estado: Pending
tipo: software-test
ejecutable_en: 1 sesion
---
# T001: graph_test.go — ciclos 4 nodos, disjuntos, multiples broken links

**Story**: [S002 Internal Edge Cases](README.md)

## Contexto

`internal/graph/graph_test.go` testea ciclos de 3 nodos y auto-referencias, pero no cubre: ciclos con 4+ nodos (A→B→C→D→A), multiples ciclos disjuntos en el mismo grafo, y un nodo con multiples links rotos. Estos son escenarios reales en proyectos con dependencias complejas.

## Alcance

**In**: Agregar a `internal/graph/graph_test.go`:
1. `TestDetectCycles_FourNodeCycle` — A→B→C→D→A: DetectCycles retorna 1 ciclo con 4 nodos unicos
2. `TestDetectCycles_MultipleDisjoint` — dos ciclos A→B→A y C→D→C en el mismo grafo: DetectCycles retorna 2 ciclos
3. `TestBrokenLinks_MultipleFromSameSource` — nodo A con links a X, Y, Z donde X e Y no existen: BrokenLinks retorna 2 broken links con Source=="a.md"
4. `TestBuild_EmptyGraph` — Build([]) retorna grafo con 0 nodos, 0 edges, DetectCycles=[], BrokenLinks=[]

**Out**: Cambios a graph.go, tests de CLI

## Estado inicial esperado

- `internal/graph/graph_test.go` existe con tests de 3-nodo cycle, self-reference, broken links
- `go test ./internal/graph/ -race` pasa

## Criterios de Aceptacion

- `go test ./internal/graph/ -run TestDetectCycles_FourNode -v` pasa
- `go test ./internal/graph/ -run TestDetectCycles_MultipleDisjoint -v` pasa
- `go test ./internal/graph/ -run TestBrokenLinks_MultipleFromSameSource -v` pasa
- `go test ./internal/graph/ -run TestBuild_EmptyGraph -v` pasa
- `go test ./internal/graph/ -race` pasa sin regresiones

## Fuente de verdad

- `internal/graph/graph_test.go` — archivo a extender
- `internal/graph/graph.go` — implementacion de Build, DetectCycles, BrokenLinks
