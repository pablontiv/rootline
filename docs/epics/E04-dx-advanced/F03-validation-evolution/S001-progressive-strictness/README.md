# S001: Progressive Strictness

**Feature**: [F03 Validation Evolution](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Cada campo y regla de validacion en .stem tiene un nivel de severidad configurable (error, warn, off) que determina su impacto en el exit code

## Antes / Despues

**Antes**: La validacion es binaria — un campo required faltante causa exit code 1 sin distincion. Agregar un nuevo campo required a un .stem rompe todos los archivos existentes inmediatamente. No hay forma de introducir reglas gradualmente.

**Despues**: Cada campo en schema y cada regla en validate puede tener `severity: error|warn|off`. Por default es `error` (backward-compatible). Warnings se reportan pero no afectan exit code. `--strict` hace que warnings tambien fallen. En herencia, un hijo puede tighten (warn->error) pero no loosear (error->warn). Esto permite adopcion incremental: primero warn en todo, luego error campo por campo.

## Criterios de Aceptacion (semanticos)

- [ ] `severity: warn` en un campo required reporta warning sin fallar exit code
- [ ] `severity: off` en un campo required lo ignora completamente
- [ ] `--strict` hace que warnings tambien fallen con exit code 1
- [ ] Herencia respeta tighten-only (hijo puede warn->error, no error->warn)

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-severity-schema-field.md) | Agregar campo Severity a SchemaField y ValidationRule |
| [T002](T002-strict-mode-exit-codes.md) | Implementar filtrado por severity y flag --strict |

## Fuente de verdad

- `internal/rules/rules.go` — SchemaField, ValidationRule structs
- `internal/rules/validate.go` — Validate function
- `internal/rules/result.go` — ValidationResult
- `internal/rules/merge.go` — MergeStemFiles (severity inheritance)
