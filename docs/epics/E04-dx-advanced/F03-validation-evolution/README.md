# F03: Validation Evolution

**Epic**: [E04](../README.md)
**Objetivo**: La validacion soporta severidad granular (error/warn/off) para adopcion incremental y se ejecuta automaticamente via git hooks
**Beneficio**: Equipos adoptan rootline gradualmente (warn primero, error despues) y documentos invalidos nunca llegan al repositorio
**Milestone**: `.stem` con `severity: warn` genera warnings sin fallar exit code, `--strict` falla en warnings, `rootline hooks install` configura pre-commit

## Scope

**In**: Severity levels en schema fields y validation rules, flag --strict, git pre-commit hooks, staged-only validation
**Out**: Pre-push hooks, commit-msg validation, severity inheritance restrictions beyond tighten/loosear, custom hook scripts

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Progressive Strictness](S001-progressive-strictness/) | Campos y reglas tienen severity configurable con exit codes apropiados |
| S002 | [Git Integration](S002-git-integration/) | Validacion automatica en pre-commit sobre archivos staged |

## Dependencias

- Ninguna (extiende validacion existente de E03)

## Fuente de verdad

- `internal/rules/rules.go` — StemFile, SchemaField, ValidationRule
- `internal/rules/validate.go` — Validate function
- `internal/rules/result.go` — ValidationResult, BatchValidationResult
- `cmd/rootline/validate.go` — validate command
