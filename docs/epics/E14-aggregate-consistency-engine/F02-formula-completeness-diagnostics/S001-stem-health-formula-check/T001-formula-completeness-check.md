---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Add stem-health diagnostic for aggregate formula completeness

**Story**: [S001 Stem-Health Formula Check](README.md)
**Contribuye a**: Formula missing Obsolete produces warning

## Preserva

- INV1: Existing 7 stem-health checks unchanged
  - Verificar: `go test ./internal/rules/ -run TestStemHealth -race`

## Contexto

The `docs/epics/.stem` aggregate formula for `estado` handles 5 values (Completed, Blocked, On Hold, In Progress, Specified) but the `estado` enum may include additional values like `Obsolete`. When the formula doesn't cover all enum values, it silently produces the default fallback (`Pending`). This diagnostic catches the gap at schema validation time.

The existing stem-health system in `internal/rules/stem_health.go` has 7 checks. This adds check #8: `aggregate-formula-coverage`.

## Alcance

**In**:
1. Add new check function in `internal/rules/stem_health.go`
2. For each field in `aggregate:`, if the field has `type: enum` in schema: extract all quoted string literals from the aggregate expression using regex, compare against `schema[field].Values` (enum values)
3. If any enum value is not found as a quoted string in the expression -> add health warning
4. Register the check in the stem-health check list

**Out**: Auto-fixing formulas, formula migration, changes to aggregate computation

## Estado inicial esperado

- `internal/rules/stem_health.go` exists with 7 checks and `ValidateStemHealth` function
- `docs/epics/.stem` has aggregate expression for estado

## Criterios de Aceptacion

- New check `aggregate-formula-coverage` exists in stem-health
- `rootline validate --all --stem-health docs/epics/` reports warning about Obsolete not covered
- Existing 7 checks still work unchanged
- `go test ./internal/rules/ -race` passes

## Fuente de verdad

- `internal/rules/stem_health.go` — existing checks, ValidateStemHealth
- `docs/epics/.stem` — aggregate expressions (L60-66)
