---
estado: Pending
tipo: software-test
ejecutable_en: 1 sesion
---
# T003: expr_eval_test.go — type mismatch, boolean literals, campo ausente

**Story**: [S002 Internal Edge Cases](README.md)

## Contexto

El evaluador de expresiones en `internal/query/expr_eval.go` soporta `==`, `!=`, `in`, `contains`, y `&&`. Los tests existentes cubren los happy paths pero no: comparacion de campo numerico con string literal (type mismatch), uso de `true`/`false` como literals en expresiones, y el comportamiento cuando un campo no existe en el record.

## Alcance

**In**: Agregar a `internal/query/expr_eval_test.go` (o el archivo de tests existente del package query):
1. `TestExprEval_TypeMismatch_NumericFieldStringCompare` — record con `version: 1` (int), expresion `version == 'texto'` → retorna false (no panic)
2. `TestExprEval_BooleanLiteral_True` — record con `active: true` (bool YAML), expresion `active == true` → retorna true
3. `TestExprEval_BooleanLiteral_False` — `active == false` cuando `active: true` → retorna false
4. `TestExprEval_FieldAbsent_NilCheck` — record sin campo `estado`, expresion `estado != nil` → retorna false (no error)
5. `TestExprEval_FieldAbsent_EqCheck` — record sin `estado`, expresion `estado == 'Pending'` → retorna false (no panic)

**Out**: Cambios a expr_eval.go, nuevos operadores

## Estado inicial esperado

- `internal/query/` existe con tests de expr eval
- `go test ./internal/query/ -race` pasa

## Criterios de Aceptacion

- Los 5 tests pasan sin panics ni errores inesperados
- El comportamiento ante type mismatch es retornar false, no crashear
- `go test ./internal/query/ -race` pasa sin regresiones

## Fuente de verdad

- `internal/query/expr_eval.go` — evaluador de expresiones
- `internal/query/` — directorio con tests existentes a identificar y extender
