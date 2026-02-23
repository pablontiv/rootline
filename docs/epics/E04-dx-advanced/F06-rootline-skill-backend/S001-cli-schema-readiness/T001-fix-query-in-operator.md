---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Fix query `in` operator — StringSliceVar a StringArrayVar

**Story**: [S001 CLI & Schema Readiness](README.md)

## Contexto

Cobra `StringSliceVar` splitea automaticamente por coma al parsear flags. Esto rompe el operador `in` de rootline query, que espera recibir la expresion completa `"estado in Pending,Especificado"` como un solo string. `StringArrayVar` de Cobra no splitea por coma — cada `--where` flag se recibe intacta.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: queryCmd flags
    metodos:
      - nombre: StringArrayVar (reemplaza StringSliceVar)
        input: "&queryWhere, where, nil, help"
        output: "flag registrada sin split por coma"
dependencias_externas: []
tests:
  - --where "estado in Pending,Especificado" retorna records con ambos estados
  - --where "estado eq Pending" sigue funcionando (no regression)
  - Multiples --where flags funcionan (AND logic)
```

## Dependencias

- Ninguna

## Alcance

**In**:
1. Cambiar `StringSliceVar` a `StringArrayVar` en `cmd/rootline/query.go:31`
2. Actualizar tests existentes si los hay
3. Verificar que multiples `--where` flags siguen funcionando

**Out**: Cambios al parser de expresiones, nuevos operadores, cambios a query engine

## Estado inicial esperado

- `cmd/rootline/query.go` existe con `StringSliceVar` en linea 31
- Tests de query existen en `cmd/rootline/commands_test.go` o similar

## Criterios de Aceptacion

- `rootline query docs/epics/ --where "estado in Pending,Especificado" --output table` retorna records sin error
- `rootline query docs/epics/ --where "estado eq Pending" --output table` sigue funcionando
- `rootline query docs/epics/ --where "estado eq Pending" --where "tipo eq software-module" --output table` retorna solo software-modules pendientes
- `go test ./cmd/rootline/ -run TestQuery -race` pasa

## Fuente de verdad

- `cmd/rootline/query.go:31` — flag definition a cambiar
- `internal/query/` — query engine (no deberia necesitar cambios)
