---
estado: Specified
tipo: refactor
ejecutable_en: 1 sesion
---
# T002: Eliminar campo Category del struct Inference

**Story**: [S004 Naming Remediation](README.md)
**Contribuye a**: Struct Inference no tiene campo `Category int`

[[blocks:T001-rename-go-files]]

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage >=85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

El struct `Inference` en `internal/infer/inference.go` tiene un campo `Category int` con JSON tag `"category"`. Este campo es redundante porque `Type string` ya identifica univocamente cada inferencia (ej: `"constant_field"`, `"link_type_violation"`, `"missing_back_reference"`). El campo Category viene de la clasificacion interna de investigacion (Cat 5, Cat 7, etc.) y no aporta informacion al consumidor. Auditoria confirmo que nada fuera de `internal/infer/` referencia este campo.

## Dependencias

> T001 debe completarse primero — los archivos ya estaran renombrados a nombres descriptivos.

## Alcance

**In**:
1. Eliminar `Category int json:"category"` del struct Inference en `inference.go`
2. Eliminar todas las asignaciones `Category: N` en los 4 archivos de detectores:
   - `link_validation.go`: 2 asignaciones (lineas con `Category: 5`)
   - `back_references.go`: 1 asignacion (`Category: 7`)
   - `constant_fields.go`: 1 asignacion (`Category: 8`)
   - `cross_references.go`: 2 asignaciones (`Category: 10`)
3. Eliminar todas las aserciones `.Category` en tests:
   - `link_validation_test.go`: 2 aserciones (`got[0].Category != 5`)
   - `constant_fields_test.go`: 1 asercion (`got[0].Category != 8`)
   - `cross_references_test.go`: 1 asercion (`got[0].Category != 10`)

**Out**: No modificar logica de deteccion. No cambiar el campo `Type string` (ese es correcto). No actualizar docs del roadmap (eso es T003).

## Estado inicial esperado

- T001 completado — archivos ya renombrados a nombres descriptivos
- Struct Inference tiene campo `Category int json:"category"`
- 6 asignaciones `Category:` en archivos de detectores
- 4+ aserciones `.Category` en archivos de test

## Criterios de Aceptacion

- `grep "Category" internal/infer/inference.go | wc -l` retorna 0
- `grep "Category:" internal/infer/link_validation.go internal/infer/back_references.go internal/infer/constant_fields.go internal/infer/cross_references.go | wc -l` retorna 0
- `grep "\.Category" internal/infer/*_test.go | wc -l` retorna 0
- `go vet ./internal/infer/` sin errores
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/infer/inference.go` — struct a modificar
- `internal/infer/link_validation.go`, `back_references.go`, `constant_fields.go`, `cross_references.go` — asignaciones a eliminar
- `internal/infer/*_test.go` — aserciones a eliminar
