---
ejecutable_en: 1 sesion
estado: Specified
tipo: software-module
---
# T004: Tests para categorias 6/12/13

**Story**: [S002 Body-Aware Categories 6/12/13](README.md)
**Contribuye a**: Cobertura completa de detectores body-aware

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

T001-T003 implementan detectores body-aware para cats 6/12/13. Este task añade fixtures con documentos reales y edge cases especificos de body parsing: markdown malformado, secciones vacias, invariantes en formatos no estandar.

## Alcance

**In**:
1. Fixtures markdown en `internal/infer/testdata/categories/body-aware/`
2. Cat 6: Fixture con 5 docs con secciones consistentes + 1 doc con seccion extra
3. Cat 12: Fixture con invariantes en seccion Invariantes y Preserva
4. Cat 13: Fixture con 3 tipos distintos y campos exclusivos por tipo
5. Edge cases: body sin headings, invariante sin ID numerico, tipo con 1 solo record

**Out**: Tests de integracion end-to-end con analyze command (F03).

## Estado inicial esperado

- T001-T003 completados — detectores de cats 6/12/13 existen
- F01 completado — AST disponible

## Criterios de Aceptacion

- ≥3 fixtures creados en testdata/categories/body-aware/
- Test de integracion para cada categoria (6, 12, 13) usando fixtures
- Edge case: body sin headings → cat 6 retorna []Inference vacio (no panic)
- Edge case: `INV:` sin numero → cat 12 no lo extrae
- Coverage de detectores body-aware ≥85%
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/infer/` — detectores bajo test
- `internal/infer/testdata/` — fixtures
