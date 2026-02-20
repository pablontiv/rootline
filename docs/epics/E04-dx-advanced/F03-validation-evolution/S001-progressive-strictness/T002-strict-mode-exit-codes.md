---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar filtrado por severity y flag --strict en validate

**Story**: [S001 Progressive Strictness](README.md)

## Contexto

Con T001, ValidationError ya lleva severity. Ahora se necesita: (1) que Validate() propague la severity del SchemaField/ValidationRule al ValidationError, (2) que ValidationResult/BatchValidationResult separen errors de warnings, (3) que el comando validate use solo errors para exit code (salvo --strict que incluye warnings).

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules + cmd/rootline
interfaces:
  - nombre: ValidationError
    metodos:
      - nombre: Severity
        input: ""
        output: "string"
dependencias_externas: []
tests:
  - Validate con campo severity=warn genera ValidationError con severity=warn
  - ValidationResult.Errors solo contiene severity=error
  - ValidationResult.Warnings contiene severity=warn
  - validate sin --strict exit code 0 con solo warnings
  - validate con --strict exit code 1 con warnings
  - validate con errors exit code 1 siempre
```

## Dependencias

- T001 (Severity field en SchemaField y ValidationRule)

## Alcance

**In**:
1. Agregar `Severity string` a `ValidationError` en result.go
2. En Validate(): propagar severity de SchemaField/ValidationRule al error generado
3. En ValidationResult: separar Errors (severity=error) y Warnings (severity=warn)
4. En BatchValidationResult: separar conteos de errors y warnings en Summary
5. En cmd/rootline/validate.go: flag `--strict`
6. Exit code: 1 si hay errors, o si --strict y hay warnings
7. JSON output incluye severity en cada error

**Out**: Severity en describe output, severity en query results, progressive migration tools

## Estado inicial esperado

- T001 completado (Severity en SchemaField, ValidationRule)
- ValidationError, ValidationResult structs existentes

## Criterios de Aceptacion

- `rootline validate` con campo severity=warn reporta warning sin exit code 1
- `rootline validate --strict` con campo severity=warn retorna exit code 1
- `rootline validate` con campo severity=error retorna exit code 1
- `rootline validate -o json` incluye "severity" en cada error del output
- JSON output de batch incluye "errors_count" y "warnings_count" en summary
- Tests existentes de validate siguen pasando
- `go test ./... -race` pasa

## Fuente de verdad

- `internal/rules/validate.go` — Validate function
- `internal/rules/result.go` — ValidationError, ValidationResult, BatchValidationResult
- `cmd/rootline/validate.go` — validate command, exit codes
