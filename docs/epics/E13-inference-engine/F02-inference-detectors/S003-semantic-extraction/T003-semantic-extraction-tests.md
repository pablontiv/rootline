---
estado: Specified
tipo: test
ejecutable_en: 1 sesion
---
# T003: Tests para detectores de formal dependency y traceability

**Story**: [S003 Semantic Extraction](README.md)
**Contribuye a**: Cobertura completa de detectores de extracción semántica

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

T001-T002 implementan los detectores de formal dependency y traceability. Este task añade fixtures con documentos reales y valida que las inferencias con `requires_agent: true` estan correctamente flagged.

## Alcance

**In**:
1. Fixtures en `internal/infer/testdata/fixtures/semantic/`
2. Formal dependency: Documento con wiki-links `[[blocks:X]]` y seccion Dependencias con items informales
3. Traceability: Documento con `**Contribuye a**:` y `**Satisface**:` con targets variados
4. Edge cases: seccion Dependencias vacia, traceability sin target, multiples claims en 1 linea
5. Validar que `requires_agent` flag es consistente

**Out**: Tests con agent real (agent Epic). Tests end-to-end con analyze (F03).

## Estado inicial esperado

- T001-T002 completados — detectores de formal dependency y traceability existen

## Criterios de Aceptacion

- ≥2 fixtures creados en testdata/fixtures/semantic/
- Test: formal dependency (wiki-link) tiene `requires_agent: false`
- Test: informal dependency (prosa) tiene `requires_agent: true`
- Test: traceability con wiki-link target tiene `requires_agent: false`
- Edge case: body sin patrones de traceability → retorna []Inference vacio
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/infer/` — detectores bajo test
- `internal/infer/testdata/` — fixtures
