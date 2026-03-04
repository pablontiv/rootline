---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Añadir --incremental flag a analyze

**Story**: [S002 Incremental Analysis Mode](README.md)
**Contribuye a**: `rootline analyze --incremental` reporta solo deltas

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`

## Contexto

T001 implementa FilterCoveredInferences. Este task integra esa funcion en el comando analyze via flag --incremental.

## Alcance

**In**:
1. Añadir `--incremental` flag al comando analyze en analyze.go
2. Cuando flag presente: cargar .stem, ejecutar detectores, filtrar con FilterCoveredInferences
3. Report incluye campo `incremental: true` cuando es incremental
4. Sin flag: comportamiento actual (full analysis)

**Out**: UI especifica para mostrar deltas vs full (JSON es suficiente).

## Estado inicial esperado

- T001 completado (FilterCoveredInferences existe)
- analyze command funcional (S001 completado)

## Criterios de Aceptacion

- `rootline analyze --incremental docs/epics/` produce report con solo deltas
- Report incremental tiene `incremental: true` en JSON
- Sin --incremental → report completo sin campo incremental
- `go test ./... -race` pasa verde

## Fuente de verdad

- `cmd/rootline/analyze.go` — comando analyze
- `internal/infer/` — FilterCoveredInferences
