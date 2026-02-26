---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T004: Fix mergeFieldSeverity empty severity default

**Story**: [S001 Level Parsing and Expansion](README.md)
**Contribuye a**: ResolveForRecord produce effective schema correcto — severity de campos expandidos desde levels refleja la intención real

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV3: `.stem` sin `levels:` zero regression
  - Verificar: `go test ./internal/rules/ -run TestMerge -v`

## Contexto

Cuando un `.stem` file define un campo en `levels.task.schema` con `required: true` pero sin `severity` explicito, el valor `Severity=""` entra al merge engine. `mergeFieldSeverity` usa `severityOrder[""]` que retorna 0 (mismo que "off"). Si el nivel padre (feature) define el mismo campo con `severity: warn`, el merge cree que el child intenta "aflojar" y mantiene `warn`. El resultado es que campos required en task level generan warnings en vez de errors.

## Especificacion Tecnica

En `internal/rules/merge.go`, funcion `mergeFieldSeverity`:

```go
// ANTES (bug):
func mergeFieldSeverity(parent, child SchemaField) SchemaField {
    parentSev := severityOrder[parent.Severity]
    childSev := severityOrder[child.Severity]  // "" → 0 (= "off")
    if childSev < parentSev {
        child.Severity = parent.Severity  // keeps "warn" instead of "error"
    }
    return child
}

// DESPUES (fix):
func mergeFieldSeverity(parent, child SchemaField) SchemaField {
    if child.Severity == "" {
        child.Severity = "error"  // unspecified defaults to "error"
    }
    // ... rest unchanged
}
```

## Alcance

**In**:
1. Agregar default `child.Severity = "error"` cuando es empty string en `mergeFieldSeverity`
2. Agregar test en `severity_test.go` para merge parent=warn + child="" → result=error

**Out**: Cambios al `severityOrder` map, cambios a `ParseStem`

## Estado inicial esperado

- `internal/rules/merge.go` tiene `mergeFieldSeverity` sin default para empty severity
- `severity_test.go` tiene tests para tighten/loosen pero no para empty severity

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestMergeSeverityEmptyDefaultsToError -v` pasa
- `go test ./... -race` pasa sin regresiones
- Binary reconstruido reporta `tipo` faltante como `severity: "error"` (no "warn")

## Fuente de verdad

- `internal/rules/merge.go:88-98` — mergeFieldSeverity function
- `internal/rules/severity_test.go` — severity merge tests
- `internal/rules/rules.go:149-153` — severityOrder map
