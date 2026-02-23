---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T005: Agregar --where flag a graph command

**Story**: [S001 Core Implementation](README.md)

[[blocks:T001-shared-filter-helper]]

## Contexto

El comando `graph` en `cmd/rootline/graph.go` hace scan → extraer wiki-links → construir grafo → render (DOT/Mermaid/check). El filtrado debe insertarse despues de scan, antes de construccion del grafo, usando `filterRecords()` de T001. Esto permite generar grafos de dependencia de un subset (ej: solo tasks no-feature).

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: graphCmd
    metodos:
      - nombre: runGraph
        input: "cmd *cobra.Command, args []string"
        output: error
dependencias_externas: []
tests:
  - graph --where filtra records antes de construir grafo
  - graph --where con expresion invalida retorna error
  - graph sin --where funciona igual que antes
  - graph --check --where reporta solo ciclos/broken links del subset filtrado
```

## Dependencias

- T001 completado (filterRecords helper disponible)

## Alcance

**In**:
1. Agregar flag `--where` (StringArrayVar) al graph command
2. Llamar `filterRecords(records, wheres)` despues de scan, antes de graph build
3. Agregar tests en `graph_test.go` para --where filtering

**Out**: Cambios a internal/graph/, documentacion (S002)

## Estado inicial esperado

- T001 completado (filter.go con filterRecords)
- `cmd/rootline/graph.go` sin --where flag

## Criterios de Aceptacion

- `rootline graph docs/epics/ --where "tipo != 'feature'" --check` solo analiza records no-feature
- `rootline graph docs/epics/ --where "estado == 'Specified'" -o dot` genera DOT solo con records Specified
- `go test ./cmd/rootline/ -run TestGraph -v` pasa
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `cmd/rootline/graph.go`
- `cmd/rootline/graph_test.go`
- `cmd/rootline/filter.go` (T001)
