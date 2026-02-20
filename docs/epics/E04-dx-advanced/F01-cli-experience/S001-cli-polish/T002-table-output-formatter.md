---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar table output formatter y aplicar a validate, query, describe

**Story**: [S001 CLI Polish](README.md)

## Contexto

Stats ya tiene `renderStatsTable()` como referencia. Se necesita un helper compartido que formatee datos tabulares con `text/tabwriter` (stdlib) y aplicarlo a los 3 comandos restantes: validate (batch), query, y describe. Tree ya tiene output ASCII propio. El flag `--output table` ya existe como global flag en root.go.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: tableWriter
    metodos:
      - nombre: renderTable
        input: "w io.Writer, headers []string, rows [][]string"
        output: "void"
dependencias_externas: []
tests:
  - renderTable con headers y rows produce output alineado
  - validate --all -o table muestra tabla con File, Valid, Errors
  - query -o table muestra registros en columnas
  - describe -o table muestra schema fields con Type, Required, Source
```

## Dependencias

- T001 no es bloqueante (independiente)
- Comandos validate, query, describe existentes de E03

## Alcance

**In**:
1. Helper `renderTable(w io.Writer, headers []string, rows [][]string)` en `cmd/rootline/table.go`
2. Validate batch: tabla con columnas File, Valid, Errors
3. Query: tabla con columnas Path + campos de frontmatter presentes
4. Describe: tabla con columnas Field, Type, Required, Values, Source

**Out**: Colores, unicode box-drawing, paginacion, CSV output

## Estado inicial esperado

- `outputFormat` flag funcional (json|table) en root.go
- Stats `renderStatsTable()` como patron de referencia
- Comandos validate, query, describe produciendo JSON

## Criterios de Aceptacion

- `rootline validate --all -o table` contra docs/epics/ muestra tabla con al menos 3 columnas
- `rootline query --from docs/epics -o table` muestra registros en columnas alineadas
- `rootline describe docs/epics -o table` muestra schema fields tabulados
- `rootline stats -o table` sigue funcionando (no se rompe)
- `-o json` sigue siendo el default y funciona sin cambios
- `go test ./cmd/rootline/ -race` pasa

## Fuente de verdad

- `cmd/rootline/stats.go` — renderStatsTable() como referencia
- `cmd/rootline/root.go` — outputFormat flag
- `cmd/rootline/validate.go`, `query.go`, `describe.go` — archivos a modificar
