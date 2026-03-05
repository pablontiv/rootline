---
estado: Completed
tipo: test
ejecutable_en: 1 sesion
---
# T003: Tests de analisis incremental

**Story**: [S002 Incremental Analysis Mode](README.md)
**Contribuye a**: Modo incremental validado end-to-end

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

El modo incremental debe producir menos inferencias que el full analysis para directorios con .stem bien definidos. Necesita fixtures que tengan .stem parcialmente completo para generar deltas predecibles.

## Alcance

**In**:
1. Fixture en testdata con .stem que cubre 50% de las inferencias posibles
2. Test: full analysis → N inferencias; incremental → <N inferencias
3. Test: .stem que cubre 100% → incremental produce 0 inferencias
4. Test: directorio sin .stem → incremental = full (nada que filtrar)

**Out**: Tests de apply (S003).

## Estado inicial esperado

- S002/T001-T002 completados
- Fixtures de analyze (S001/T004) disponibles como base

## Criterios de Aceptacion

- Test: incremental filtra inferencias cubiertas
- Test: 0 deltas cuando .stem esta completo
- Test: incremental sin .stem produce mismo resultado que full
- `go test ./... -race` pasa verde

## Fuente de verdad

- `internal/e2e/testdata/analyze/` — fixtures
- `internal/infer/` — FilterCoveredInferences
