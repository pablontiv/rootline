---
ejecutable_en: 1 sesion
estado: Specified
tipo: software-module
---
# T004: Tests de integracion para analyze

**Story**: [S001 Analyze Command & Report Format](README.md)
**Contribuye a**: Analyze command validado end-to-end

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

El comando analyze orquesta todos los detectores. Tests de integracion validan el pipeline completo: directorio con .stem + documentos → report JSON con inferencias correctas.

## Alcance

**In**:
1. Fixture directory en `internal/e2e/testdata/analyze/` con .stem y documentos
2. Test: analyze produce report JSON parseable con version: 1
3. Test: analyze con directorio vacio produce report con 0 inferencias
4. Test: analyze con --output table produce output sin panic
5. Test: inferencias de constant-field aparecen en report cuando fixture tiene constantes

**Out**: Tests de apply (S003). Tests de modo incremental (S002).

## Estado inicial esperado

- T001-T003 completados (analyze command funcional)
- internal/e2e/testdata/ existe para otros tests e2e

## Criterios de Aceptacion

- ≥3 test cases en e2e
- Test: JSON report tiene `version: 1` y `kind: "analyze"`
- Test: directorio vacio → report valido con detectores sin resultados
- Test: fixture con campo constante → constant-field detector produce inferencia
- `go test ./internal/e2e/ -race` pasa verde

## Fuente de verdad

- `internal/e2e/` — tests e2e existentes como referencia
- `cmd/rootline/analyze.go` — comando bajo test
