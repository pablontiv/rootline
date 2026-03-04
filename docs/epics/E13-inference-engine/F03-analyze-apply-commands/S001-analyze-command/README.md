---
estado: Specified
tipo: historia
---
# S001: Analyze Command & Report Format

**Feature**: [F03 Analyze & Apply Commands](../README.md)
**Capacidad**: `rootline analyze` orquesta todos los detectores y genera report JSON
**Cubre**: Milestone de F03 — analyze genera report con version: 1

## Antes / Despues

**Antes**: No existe comando `analyze`. Los detectores de inferencia (cats 1-13) existen como funciones individuales pero no hay orquestacion ni formato de reporte unificado.

**Despues**: `rootline analyze <path>` ejecuta todos los detectores, produce JSON report con `version: 1`, `kind: "analyze"`, inferencias agrupadas por categoria, y soporte para `--output table`.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline analyze docs/epics/E13-inference-engine/` produce JSON con todas las categorias
- [ ] Report tiene estructura: `{version: 1, kind: "analyze", categories: [{id, name, inferences}]}`
- [ ] `--output table` produce tabla legible con resumen por categoria
- [ ] `--output json` produce JSON valido
- [ ] `go test ./... -race` pasa verde

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV2: Contratos JSON mantienen `"version": 1`
  - Verificar: Report incluye `"version": 1`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-analyze-report-schema.md) | Definir analyze report JSON schema |
| [T002](T002-analyze-command.md) | Implementar comando rootline analyze |
| [T003](T003-analyze-table-output.md) | Añadir table output mode para analyze |
| [T004](T004-analyze-integration-tests.md) | Tests de integracion para analyze |

## Fuente de verdad

- `cmd/rootline/` — subcomandos CLI
- `internal/infer/` — detectores de categorias
- Patron existente: `cmd/rootline/validate.go` como referencia de comando similar
