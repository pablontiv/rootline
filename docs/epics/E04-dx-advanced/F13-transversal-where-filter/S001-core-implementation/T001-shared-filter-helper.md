---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Crear shared filterRecords() helper y refactor query

**Story**: [S001 Core Implementation](README.md)

## Contexto

`rootline query` es el unico comando con filtrado `--where`. La infraestructura ya existe en `internal/query/expr_eval.go`: `CompileWhere()` compila expresiones, `BuildEnv()` construye el environment de un record, y `MatchRecord()` evalua si un record matchea. Para reutilizar esto en tree/stats/validate/graph, se necesita un helper compartido en `cmd/rootline/` que encapsule el patron comun: compilar N expresiones con AND, filtrar un slice de records.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: filterRecords
    metodos:
      - nombre: filterRecords
        input: "records []*extract.Record, wheres []string"
        output: "[]*extract.Record, error"
dependencias_externas: []
tests:
  - filterRecords con match retorna subset correcto
  - filterRecords sin match retorna slice vacio
  - filterRecords con expresion invalida retorna error
  - filterRecords con wheres vacio retorna todos los records (passthrough)
  - filterRecords con multiples wheres los combina con AND
```

## Dependencias

- `internal/query/expr_eval.go` existente con CompileWhere, BuildEnv, MatchRecord

## Alcance

**In**:
1. Crear `cmd/rootline/filter.go` con funcion `filterRecords(records []*extract.Record, wheres []string) ([]*extract.Record, error)`
2. La funcion une wheres con ` && `, compila con `query.CompileWhere()`, itera records con `query.MatchRecord()`
3. Crear `cmd/rootline/filter_test.go` con unit tests
4. Refactorizar `cmd/rootline/query.go` para usar `filterRecords()` en vez de su implementacion inline

**Out**: Integracion en tree/stats/validate/graph (T002-T005), cambios a internal/query/

## Estado inicial esperado

- `internal/query/expr_eval.go` con CompileWhere, BuildEnv, MatchRecord funcionales
- `cmd/rootline/query.go` con implementacion inline de filtrado --where

## Criterios de Aceptacion

- `go build ./cmd/rootline/` compila sin errores
- `go test ./cmd/rootline/ -run TestFilter -v` pasa todos los tests
- `go test ./cmd/rootline/ -run TestQuery -v` pasa (query refactorizado sin regresion)
- `rootline query docs/epics/ --where "estado == 'Completed'" -o json` retorna mismo resultado que antes del refactor

## Fuente de verdad

- `internal/query/expr_eval.go` (CompileWhere, BuildEnv, MatchRecord)
- `cmd/rootline/query.go` (implementacion actual de --where)
