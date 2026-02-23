---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Construir grafo dirigido con deteccion de ciclos y links rotos

**Story**: [S003 Graph Command](README.md)

## Contexto

Con links extraidos en cada Record, se puede construir un grafo dirigido donde los nodos son documentos y los edges son links tipados. El grafo permite detectar ciclos (dependencias circulares) y links rotos (target que no corresponde a ningun documento). Esto es la logica pura — el comando CLI (T002) la consume.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/graph
interfaces:
  - nombre: Graph
    metodos:
      - nombre: Build
        input: "records []*extract.Record"
        output: "*Graph"
      - nombre: DetectCycles
        input: ""
        output: "[][]string"
      - nombre: BrokenLinks
        input: ""
        output: "[]BrokenLink"
dependencias_externas: []
tests:
  - Graph con 3 records sin ciclos retorna cycles vacio
  - Graph con ciclo A->B->C->A detecta ciclo
  - Graph con link a target inexistente reporta broken link
  - Graph sin links retorna grafo vacio
  - Graph con self-reference detecta ciclo
```

## Dependencias

- internal/extract (Record con Links)

## Alcance

**In**:
1. Paquete `internal/graph/graph.go`
2. Struct `Graph` con nodos (records) y edges (links)
3. `Build(records)` — construir grafo desde records y sus links
4. `DetectCycles()` — DFS para encontrar ciclos, retorna lista de ciclos (cada ciclo es lista de paths)
5. `BrokenLinks()` — links cuyo target no corresponde a ningun record.Path
6. Struct `BrokenLink` con: Source (record path), Target, Type, Line
7. Tests unitarios

**Out**: Topological sort, shortest path, graph metrics, visualization

## Estado inicial esperado

- Record.Links poblado (F05/S001)
- Paquete internal/graph/ no existe (crear)

## Criterios de Aceptacion

- `Build(records)` con 3 records y links entre ellos construye grafo correcto
- `DetectCycles()` con A→B→C→A retorna `[[A,B,C,A]]`
- `DetectCycles()` sin ciclos retorna []
- `BrokenLinks()` con link a "nonexistent.md" retorna BrokenLink
- `BrokenLinks()` con todos links validos retorna []
- `go test ./internal/graph/ -race` pasa

## Fuente de verdad

- `internal/extract/extract.go` — Record, Link structs
