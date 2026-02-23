---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Agregar --where flag a stats command

**Story**: [S001 Core Implementation](README.md)

[[blocks:T001-shared-filter-helper]]

## Contexto

El comando `stats` en `cmd/rootline/stats.go` hace scan → derive → aggregate → conteo por campo → render. El filtrado debe insertarse despues de derivacion, antes de conteo, usando el helper `filterRecords()` de T001. Esto permite responder preguntas como "cuantos software-module tasks hay pendientes?".

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: statsCmd
    metodos:
      - nombre: runStats
        input: "cmd *cobra.Command, args []string"
        output: error
dependencias_externas: []
tests:
  - stats --where filtra records antes de contar
  - stats --where con expresion invalida retorna error
  - stats sin --where funciona igual que antes
```

## Dependencias

- T001 completado (filterRecords helper disponible)

## Alcance

**In**:
1. Agregar flag `--where` (StringArrayVar) al stats command
2. Llamar `filterRecords(records, wheres)` despues de derivacion, antes de conteo
3. Agregar tests en `stats_test.go` para --where filtering

**Out**: Cambios a internal/query/, documentacion (S002)

## Estado inicial esperado

- T001 completado (filter.go con filterRecords)
- `cmd/rootline/stats.go` sin --where flag

## Criterios de Aceptacion

- `rootline stats docs/epics/ --where "tipo == 'software-module'" -o table` cuenta solo software-module records
- `rootline stats docs/epics/ --where "estado == 'Completed'" -o table` cuenta solo completados
- `go test ./cmd/rootline/ -run TestStats -v` pasa
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `cmd/rootline/stats.go`
- `cmd/rootline/stats_test.go`
- `cmd/rootline/filter.go` (T001)
