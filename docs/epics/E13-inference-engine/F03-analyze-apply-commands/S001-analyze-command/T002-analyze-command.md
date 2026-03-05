---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar comando rootline analyze

**Story**: [S001 Analyze Command & Report Format](README.md)
**Contribuye a**: `rootline analyze <path>` orquesta detectores y genera report

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV2: Contratos JSON mantienen `"version": 1`
  - Verificar: `rootline analyze docs/epics/ --output json | jq .version`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Cada categoria (1-13) tiene un detector implementado en internal/infer/. El comando `analyze` orquesta todos los detectores, recoge inferencias, y produce el AnalyzeReport definido en T001.

## Alcance

**In**:
1. Crear `cmd/rootline/analyze.go` con cobra command
2. Registrar en root.go
3. Orquestar: index dir → extract records → load stems → run cada detector → collect inferencias
4. Producir AnalyzeReport JSON con --output json (default)
5. Manejar errores de extraccion/parsing sin abortar (log warning, continuar)

**Out**: Table output (T003). Modo incremental (S002). Tests de integracion (T004).

## Estado inicial esperado

- T001 completado (AnalyzeReport schema existe)
- F02 completado (detectores de cats 5-13 existen)
- Cats 1-4 detectores en Analyze() existente

## Criterios de Aceptacion

- `rootline analyze docs/epics/E13-inference-engine/` produce JSON report
- Report contiene 13 categorias con inferencias (algunas pueden estar vacias)
- `rootline analyze --help` muestra uso
- Error en 1 detector no aborta el analisis completo
- `go test ./... -race` pasa verde

## Fuente de verdad

- `cmd/rootline/validate.go` — referencia de comando similar
- `cmd/rootline/root.go` — registro de subcomandos
- `internal/infer/` — detectores
