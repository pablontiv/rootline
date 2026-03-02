---
estado: Specified
tipo: test
ejecutable_en: 1 sesion
---
# T003: Agregar test de rechazo v1 y eliminar tests backward compat

**Story**: [S001 Engine rechaza stems v1](README.md)
**Contribuye a**: No existe funcion rejectLevelsInV2 ni check "version-deprecated"

[[blocks:T002-implement-v1-rejection]]

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV2: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Despues de implementar el rechazo de v1 (T002), hay tests que prueban backward compatibility de v1 que ya no aplican. Deben eliminarse y reemplazarse con un test que verifica que v1 produce error.

## Alcance

**In**:
1. Agregar `TestParseStem_RejectsV1` en `internal/rules/rules_test.go` — verifica que version 0 y version 1 producen error con mensaje descriptivo
2. Eliminar `TestParseStemV2_V1BackwardCompat` de rules_test.go
3. Eliminar `TestParseStemV2_UnsetVersionBackwardCompat` de rules_test.go
4. Adaptar stemhealth_test.go: eliminar test del check "version-deprecated"

**Out**: No modificar logica de parsing (eso fue T002)

## Estado inicial esperado

- T002 completado: ParseStem rechaza version 0/1
- Tests de backward compat fallan (porque v1 ahora produce error)

## Criterios de Aceptacion

- `go test ./internal/rules/ -race -run TestParseStem_RejectsV1` pasa verde
- `grep -n "V1BackwardCompat\|UnsetVersionBackwardCompat" internal/rules/rules_test.go` retorna 0 resultados
- `go test ./... -race` pasa verde
- Coverage ≥85%

## Fuente de verdad

- `internal/rules/rules_test.go` — TestParseStemV2_V1BackwardCompat (lineas 463-501), TestParseStemV2_UnsetVersionBackwardCompat
- `internal/rules/stemhealth_test.go` — test de check "version-deprecated"
