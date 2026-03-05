---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Añadir table output mode para analyze

**Story**: [S001 Analyze Command & Report Format](README.md)
**Contribuye a**: `rootline analyze --output table` produce tabla legible

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`

## Contexto

Todos los comandos rootline soportan `--output table|json`. El comando analyze necesita un formato tabla que resuma inferencias por categoria: nombre, count, top inferencias.

## Alcance

**In**:
1. Implementar table renderer para AnalyzeReport en analyze.go
2. Tabla con columnas: Category | Inferences | Top Types | Requires Agent
3. Reusar helpers de `cmd/rootline/table.go` para formateo
4. `--output table` como alternativa a JSON

**Out**: Formato detallado por inferencia (JSON es suficiente para detalle).

## Estado inicial esperado

- T002 completado (analyze command existe con JSON output)
- table.go tiene helpers de formateo

## Criterios de Aceptacion

- `rootline analyze docs/epics/ --output table` produce tabla legible
- Tabla muestra 13 filas (1 por categoria) con conteo de inferencias
- Categorias sin inferencias muestran 0
- `go test ./... -race` pasa verde

## Fuente de verdad

- `cmd/rootline/table.go` — helpers de formateo de tablas
- `cmd/rootline/validate.go` — referencia de --output table
