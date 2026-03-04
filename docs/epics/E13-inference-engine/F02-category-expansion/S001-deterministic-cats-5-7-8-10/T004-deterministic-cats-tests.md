---
estado: Specified
tipo: test
ejecutable_en: 1 sesion
---
# T004: Tests para categorias 5/7/8/10

**Story**: [S001 Deterministic Categories 5/7/8/10](README.md)
**Contribuye a**: Cobertura completa de detectores deterministic cats

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

T001-T003 implementan detectores para cats 5/7/8/10. Este task añade test fixtures con documentos reales del proyecto rootline y edge cases que no se cubren en tests unitarios individuales.

## Alcance

**In**:
1. Test fixture directory en `internal/infer/testdata/categories/` con documentos markdown
2. Tests de integracion: Cat 5 con LinkSchema real (docs/epics/ .stem tiene link_schema)
3. Tests de integracion: Cat 7 con grafo real (2 docs que se referencian mutuamente)
4. Edge cases: directorio vacio, 1 solo record, campos sin valores
5. Test que todos los detectores retornan []Inference (nunca nil)

**Out**: Tests de categorias body-aware (S002/T004). Tests de integracion end-to-end (F03).

## Estado inicial esperado

- T001-T003 completados — detectores de cats 5/7/8/10 existen
- internal/infer/testdata/ existe (puede tener archivos de tests previos)

## Criterios de Aceptacion

- ≥3 test fixtures creados en testdata/categories/
- Test de integracion para cada categoria (5, 7, 8, 10) usando fixtures
- Edge case: directorio con 0 records no produce panic
- Coverage de nuevos detectores ≥85%
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/infer/` — detectores bajo test
- `internal/infer/testdata/` — fixtures existentes
