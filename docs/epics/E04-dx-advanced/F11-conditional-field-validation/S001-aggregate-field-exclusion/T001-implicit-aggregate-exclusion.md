---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Skip required check para campos con aggregate en index files

**Story**: [S001 Aggregate Field Exclusion](README.md)

## Contexto

En `internal/rules/validate.go`, la función `Validate()` itera sobre `effective.Schema` y reporta error cuando un campo tiene `Required: true` pero no existe en el frontmatter (línea ~42). No distingue si el campo es computado por agregación.

El `.stem` de `docs/epics/` declara `estado: required: true` y `aggregate.estado: <expr>`. Para index files (README.md), el valor de `estado` se computa desde los hijos via agregación — no debería exigirse en frontmatter. Esto sigue el patrón de bases de datos donde columnas `GENERATED ALWAYS AS` se excluyen automáticamente de INSERT validation.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: isIndexFile
    metodos:
      - nombre: isIndexFile
        input: "path string, stem *StemFile"
        output: "bool"
dependencias_externas: []
tests:
  - Campo con aggregate + required en index file → no ValidationError
  - Campo con aggregate + required en non-index file → sí ValidationError
  - Campo required sin aggregate en index file → sí ValidationError (comportamiento normal)
  - Index file name configurable via structural.subdirs.require_index
```

## Alcance

**In**:
1. Agregar helper `isIndexFile(path string, stem *StemFile) bool` en `validate.go` que consulta `stem.Structural.Subdirs.RequireIndex` (default "README.md")
2. En el loop de schema auto-checks, antes de reportar required error, verificar si el campo tiene `aggregate:` expression (`effective.Aggregate[name]`) Y el archivo es index file
3. Si ambas condiciones → `continue` (skip error)
4. Tests unitarios en `validate_test.go`

**Out**: Cambios al aggregate engine, cambios a derive, UI/output changes

## Estado inicial esperado

- `internal/rules/validate.go` existe con `Validate()` funcional
- `internal/rules/rules.go` tiene `StemFile` con campo `Aggregate map[string]any`
- `internal/rules/structural.go` tiene `SubdirRules` con `RequireIndex`

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestValidate -race` pasa con nuevos test cases
- Campo required + aggregate en README.md → 0 validation errors
- Campo required + aggregate en T001-xxx.md → 1 validation error (required)
- Campo required sin aggregate en README.md → 1 validation error (required)
- `go vet ./internal/rules/` sin errores
- `golangci-lint run ./internal/rules/` sin errores

## Fuente de verdad

- `internal/rules/validate.go` — función Validate(), línea ~42 (required check)
- `internal/rules/rules.go` — StemFile struct (Aggregate field), SchemaField struct
- `internal/rules/structural.go` — SubdirRules struct (RequireIndex)
- `internal/derive/aggregate.go:152-155` — isIndexFile() existente como referencia
