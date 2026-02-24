---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implement Aggregate Expression Generator

**Story**: [S001 Aggregate Expression Generator](README.md)
**Contribuye a**: GenerateAggregateExpr produce expresiones correctas EN/ES

## Preserva

- INV1: Tests existentes siguen pasando
  - Verificar: `go test ./... -race`

## Contexto

El motor de agregación de rootline (`internal/derive/`) ya evalúa expresiones `aggregate:` en `.stem` usando expr-lang/expr. Ejemplo funcional en `docs/epics/E03-rootline/.stem`:

```yaml
aggregate:
  estado: |
    all(descendants, {.estado == "Completed"}) ? "Completed" :
    any(descendants, {.estado == "Blocked"}) ? "Blocked" :
    ...
    "Pending"
```

Lo que falta es un **generador** que produzca estas expresiones automáticamente a partir de los valores de un campo enum. El generador se ubica en `internal/migrate/` porque es reutilizado por `init`, `migrate --split` y `fix`.

## Especificacion Tecnica

**Archivo**: `internal/migrate/aggregate.go`
**Tests**: `internal/migrate/aggregate_test.go`

**API pública**:
```go
func GenerateAggregateExpr(fieldName string, sf rules.SchemaField) string
func GenerateAggregates(rootSchema map[string]rules.SchemaField, existingAgg map[string]any) map[string]string
```

**Clasificación por keywords (multilingüe)**:

| Clase | Keywords | Operador | Prioridad |
|-------|----------|----------|-----------|
| Terminal | completed, completado, done, closed, obsolete, obsoleto | `all()` | Primera (condición de "todo terminado") |
| Negative | blocked, bloqueada, hold, diferida, paused | `any()` | Segunda (prioridad alta) |
| Active | in progress, en progreso, active | `any()` | Tercera |
| Neutral | (sin match) | `any()` | Última (orden original del enum) |

**Algoritmo**:
1. Si `sf.Type != "enum"` → retornar `""`
2. Clasificar cada valor del enum en una de las 4 clases usando lowercase match
3. Si 0 keywords matchean → fallback posicional: último valor = terminal, primero = default
4. Construir expresión ternaria encadenada: terminal primero (`all(descendants, ...)`), luego negativos, activos, neutrales con `any(descendants, ...)`, último valor como default

**GenerateAggregates**:
1. Para cada campo en `rootSchema` donde `sf.Type == "enum"`
2. Si `existingAgg[fieldName]` existe → skip (preservar existente)
3. Llamar `GenerateAggregateExpr` y agregar al resultado

**6 tests**:
1. Enum EN: [Pending, In Progress, Blocked, Completed] → expresión con all/any correcta
2. Enum ES: [Pending, En Progreso, Bloqueada, Diferida, Completado, Obsoleto] → clasificación multilingüe
3. Sin keywords: [Alpha, Beta, Gamma] → fallback posicional
4. 1 valor: [Done] → expresión trivial `"Done"`
5. Non-enum: tipo string → retorna ""
6. Skip existente: campo con aggregate existente → no se genera

## Dependencias

- `internal/rules` — `SchemaField` type (`rules.SchemaField{Type: "enum", Values: [...]}`)

## Alcance

**In**:
1. Crear `internal/migrate/aggregate.go` con las 2 funciones públicas + helpers privados
2. Crear `internal/migrate/aggregate_test.go` con 6 tests

**Out**: No modificar otros archivos. No integrar en CLI (eso es S002/S003).

## Estado inicial esperado

- `internal/migrate/` existe como package (contiene `rename.go`, `diff.go`)
- `internal/rules/types.go` exporta `SchemaField` con campos `Type string` y `Values []string`

## Criterios de Aceptacion

- `go test ./internal/migrate/ -run TestGenerateAggregate -v` pasa 6/6 tests
- `go vet ./internal/migrate/` sin errores
- `golangci-lint run ./internal/migrate/` sin errores

## Fuente de verdad

- `internal/rules/types.go` — SchemaField type definition
- `internal/migrate/` — package destino (ver archivos existentes para convenciones)
- `docs/epics/E03-rootline/.stem` — ejemplo de expresión aggregate funcional
