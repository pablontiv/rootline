---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Regla excludes declarativa en SchemaField

**Story**: [S001 Aggregate Field Exclusion](README.md)

[[blocks:T001-implicit-aggregate-exclusion]]

## Contexto

La capa 1 (T001) resuelve el caso más común: campos con `aggregate:` en index files. Pero existen casos donde un usuario quiere excluir `required` por otros motivos (ej: campo que solo aplica a ciertos tipos de archivo por pattern). Se necesita una regla `excludes` declarativa en `SchemaField` que permita excluir la validación `required` cuando el path del archivo matchea un glob pattern.

Esto sigue el patrón de JSON Schema `if/then/else` pero simplificado: una condición de match contra el path del archivo.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: ExcludeRule
    metodos:
      - nombre: struct fields
        input: "Match string (glob pattern)"
        output: "n/a"
dependencias_externas: []
tests:
  - SchemaField con excludes.match que matchea path → skip required
  - SchemaField con excludes.match que NO matchea path → required aplica
  - Excludes se mergea correctamente en child .stem (override)
  - YAML parsing de excludes en .stem funciona
```

## Alcance

**In**:
1. Nuevo struct `ExcludeRule` con campo `Match string` en `rules.go`
2. Agregar campo `Excludes *ExcludeRule` a `SchemaField` struct
3. En `validate.go`, después del check implícito de T001, evaluar `field.Excludes.Match` con `filepath.Match()` contra `record.Path`
4. En `merge.go`, asegurar que `Excludes` se mergea como struct (child override parent)
5. Tests unitarios

**Out**: Condiciones complejas (if/then/else), exclusión por valor de campo, UI changes

## Estado inicial esperado

- T001 completado (check implícito ya funciona)
- `internal/rules/rules.go` tiene SchemaField struct
- `internal/rules/merge.go` tiene lógica de merge de schema fields

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestValidate -race` pasa con nuevos test cases
- `go test ./internal/rules/ -run TestMerge -race` pasa con test de merge de Excludes
- YAML `.stem` con `excludes: { match: "*/README.md" }` se parsea correctamente
- Path que matchea excludes → 0 validation errors para ese campo
- Path que no matchea → validation error normal
- `golangci-lint run ./internal/rules/` sin errores

## Fuente de verdad

- `internal/rules/rules.go` — SchemaField struct (agregar ExcludeRule)
- `internal/rules/validate.go` — Validate() (agregar check después de T001)
- `internal/rules/merge.go` — merge de SchemaField
- `internal/rules/validate_test.go` — tests
