---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Agregar --where flag a tree command

**Story**: [S001 Core Implementation](README.md)

[[blocks:T001-shared-filter-helper]]

## Contexto

El comando `tree` en `cmd/rootline/tree.go` hace scan → derive → aggregate → buildTree → render. El filtrado debe insertarse despues de derivacion y antes de buildTree, usando el helper `filterRecords()` de T001. Ademas hay un bug residual de F12: linea 117 compara `estado == "Completado"` pero deberia ser `"Completed"`.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: treeCmd
    metodos:
      - nombre: runTree
        input: "cmd *cobra.Command, args []string"
        output: error
dependencias_externas: []
tests:
  - tree --where filtra records antes de construir arbol
  - tree --where con expresion invalida retorna error
  - tree sin --where funciona igual que antes
  - buildTree cuenta completados con "Completed" (no "Completado")
```

## Dependencias

- T001 completado (filterRecords helper disponible)

## Alcance

**In**:
1. Agregar flag `--where` (StringArrayVar) al tree command
2. Llamar `filterRecords(records, wheres)` despues de derivacion, antes de `buildTree`
3. Fix bug linea 117: cambiar `"Completado"` → `"Completed"`
4. Agregar tests en `tree_test.go` para --where filtering
5. Fix tests existentes que usen `"Completado"` en test data

**Out**: Cambios a internal/query/, documentacion (S002)

## Estado inicial esperado

- T001 completado (filter.go con filterRecords)
- `cmd/rootline/tree.go` sin --where flag
- Linea 117 con bug `"Completado"`

## Criterios de Aceptacion

- `rootline tree docs/epics/ --where "estado == 'Completed'" -o table` muestra solo records completados
- `rootline tree docs/epics/ --where "tipo == 'software-module'" -o table` muestra solo software-module records
- `grep '"Completado"' cmd/rootline/tree.go` retorna 0 lineas
- `go test ./cmd/rootline/ -run TestTree -v` pasa
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `cmd/rootline/tree.go`
- `cmd/rootline/tree_test.go`
- `cmd/rootline/filter.go` (T001)
