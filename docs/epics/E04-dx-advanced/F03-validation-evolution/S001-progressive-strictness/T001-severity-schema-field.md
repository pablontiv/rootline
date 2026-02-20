---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Agregar campo Severity a SchemaField y ValidationRule

**Story**: [S001 Progressive Strictness](README.md)

## Contexto

Hoy SchemaField tiene Type, Required, Values, Default, Source. Se necesita agregar Severity string con valores "error", "warn", "off". Default es "error" para backward-compatibility. En ValidationRule igualmente. En merge, severity sigue reglas type-driven (scalar replace) pero con restriccion: hijo puede tighten (warn->error) pero no loosear (error->warn).

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: SchemaField
    metodos:
      - nombre: Severity
        input: ""
        output: "string (error|warn|off)"
dependencias_externas: []
tests:
  - SchemaField sin severity explicita default a "error"
  - SchemaField con severity "warn" se parsea correctamente
  - Merge severity: hijo warn + padre error = error (no loosear)
  - Merge severity: hijo error + padre warn = error (tighten OK)
  - SchemaField con severity "off" se parsea correctamente
```

## Dependencias

- Ninguna (modifica structs existentes)

## Alcance

**In**:
1. Agregar `Severity string` a `SchemaField` en rules.go con yaml tag `severity`
2. Agregar `Severity string` a `ValidationRule` en rules.go con yaml tag `severity`
3. Default "error" si campo vacio despues de parse
4. En merge.go: al mergear SchemaFields, aplicar tighten-only rule
5. Severity ordering: off < warn < error (numerico para comparacion)
6. Tests unitarios para parse y merge de severity

**Out**: Filtrado de ValidationError por severity (T002), cambios en exit code (T002), cambios en output format

## Estado inicial esperado

- SchemaField struct en internal/rules/rules.go funcional
- MergeStemFiles en internal/rules/merge.go funcional
- Tests existentes pasan

## Criterios de Aceptacion

- `ParseStemFile` con `severity: warn` en un campo retorna SchemaField.Severity == "warn"
- `ParseStemFile` sin severity retorna SchemaField.Severity == "error" (default)
- `MergeStemFiles` con padre severity=warn y hijo severity=error retorna error (tighten)
- `MergeStemFiles` con padre severity=error y hijo severity=warn retorna error (no loosear)
- Tests existentes de merge siguen pasando
- `go test ./internal/rules/ -race` pasa

## Fuente de verdad

- `internal/rules/rules.go` — SchemaField, ValidationRule structs
- `internal/rules/merge.go` — mergeSchemaFields logic
- `internal/rules/merge_test.go` — existing merge tests
