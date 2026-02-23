---
tipo: historia
cliente: Platform Owner
---
# S001: Aggregate Field Exclusion

**Feature**: [F11 Conditional Field Validation](../README.md)
**Capacidad**: rootline auto-excluye campos con aggregate del check required en index files

## Antes / Despues

**Antes**: rootline exige `estado: required: true` en TODOS los .md incluyendo READMEs de Features/Stories/Epics. Como `estado` se computa por agregación desde hijos, los READMEs tienen estado stale (42 archivos dicen "Pending" cuando todos sus tasks son "Completado"). No hay forma de distinguir campos manuales de computados en el schema.

**Despues**: rootline detecta que un campo tiene expresión `aggregate:` y automáticamente relaja `required` en index files. Para casos no cubiertos, `excludes: { match: "*/README.md" }` permite exclusión declarativa. `rootline doctor` reporta cuando un campo es required + aggregated como warning.

## Criterios de Aceptacion (semanticos)

- [x] `rootline validate` no reporta error por `estado` faltante en README.md que tiene aggregate
- [x] `excludes` rule permite excluir required por glob pattern
- [x] `rootline doctor` detecta y reporta conflicto aggregate+required

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-implicit-aggregate-exclusion.md) | Skip required check para campos con aggregate en index files |
| [T002](T002-excludes-schema-rule.md) | Regla excludes declarativa en SchemaField |
| [T003](T003-doctor-aggregate-diagnostic.md) | Diagnóstico en doctor para aggregate+required |

## Fuente de verdad

- `internal/rules/validate.go` — punto de inserción para exclusiones
- `internal/rules/rules.go` — SchemaField y StemFile structs
- `internal/derive/aggregate.go` — isIndexFile helper existente
- `cmd/rootline/doctor.go` — patrón de checks existentes
