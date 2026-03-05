---
estado: Completed
tipo: refactor
ejecutable_en: 1 sesion
---
# T004: Fix Go code residuals — comment y testdata dir

**Story**: [S004 Naming Remediation](README.md)
**Contribuye a**: Cero terminología de investigación en código Go

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`

## Contexto

T001 renombró archivos Go de catN a nombres descriptivos. Quedaron 2 residuos: un comentario en `inference.go` que dice "category detector" y el directorio `testdata/categories/` que usa nomenclatura de investigación.

## Alcance

**In**:
1. Cambiar comentario `internal/infer/inference.go:3` de "category detector" a "inference detector"
2. Renombrar directorio `internal/infer/testdata/categories/` a `internal/infer/testdata/fixtures/`
3. Actualizar las 4 referencias `testdata/categories` en `internal/infer/integration_test.go`

**Out**: Cambios en documentos del roadmap (eso es T005/T006).

## Estado inicial esperado

- `internal/infer/inference.go` tiene comentario "category detector" en línea 3
- `internal/infer/testdata/categories/` existe con 4 fixtures .md
- `internal/infer/integration_test.go` referencia `testdata/categories` en 4 lugares

## Criterios de Aceptacion

- `grep -r "categories" internal/infer/` retorna vacío
- `grep "category" internal/infer/inference.go` retorna vacío
- `ls internal/infer/testdata/fixtures/` muestra los 4 archivos .md
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/infer/inference.go` — comentario a cambiar
- `internal/infer/testdata/categories/` — directorio a renombrar
- `internal/infer/integration_test.go` — referencias a actualizar
