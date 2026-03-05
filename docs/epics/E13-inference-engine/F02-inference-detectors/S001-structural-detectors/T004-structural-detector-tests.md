---
ejecutable_en: 1 sesion
estado: Completed
tipo: software-module
---
# T004: Tests para structural detectors

**Story**: [S001 Structural Detectors](README.md)
**Contribuye a**: Cobertura completa de detectores estructurales

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

T001-T003 implementan detectores estructurales (link-type, back-reference, constant-field, cross-reference). Este task añade test fixtures con documentos reales del proyecto rootline y edge cases que no se cubren en tests unitarios individuales.

## Alcance

**In**:
1. Test fixture directory en `internal/infer/testdata/fixtures/` con documentos markdown
2. Tests de integracion: link-type validation con LinkSchema real (docs/epics/ .stem tiene link_schema)
3. Tests de integracion: back-reference con grafo real (2 docs que se referencian mutuamente)
4. Edge cases: directorio vacio, 1 solo record, campos sin valores
5. Test que todos los detectores retornan []Inference (nunca nil)

**Out**: Tests de detectores body-aware (S002/T004). Tests de integracion end-to-end (F03).

## Estado inicial esperado

- T001-T003 completados — detectores estructurales existen
- internal/infer/testdata/ existe (puede tener archivos de tests previos)

## Criterios de Aceptacion

- ≥3 test fixtures creados en testdata/fixtures/
- Test de integracion para cada detector (link-type, back-reference, constant-field, cross-reference) usando fixtures
- Edge case: directorio con 0 records no produce panic
- Coverage de nuevos detectores ≥85%
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/infer/` — detectores bajo test
- `internal/infer/testdata/` — fixtures existentes
