---
tipo: feature
---
# F11: Conditional Field Validation

**Epic**: [E04](../README.md)
**Objetivo**: rootline distingue campos manuales de campos computados en validación required
**Beneficio**: Elimina 42+ READMEs con estado stale; campos con aggregate no necesitan estar en frontmatter de index files
**Milestone**: `rootline validate --all` pasa sin exigir campos agregados en index files; `rootline doctor` detecta conflictos aggregate+required

## Scope

**In**: Exclusión implícita de required para campos con aggregate en index files, regla `excludes` declarativa en SchemaField, diagnóstico en doctor
**Out**: Rewrite automático de frontmatter existente, cambios al aggregate engine, propagación de estado

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Aggregate Field Exclusion](S001-aggregate-field-exclusion/) | Campos con aggregate se excluyen de required en index files |

## Dependencias

- F10 completado (proposal fix engine — mismo módulo validate.go)

## Fuente de verdad

- `internal/rules/validate.go` — motor de validación
- `internal/rules/rules.go` — SchemaField struct
- `cmd/rootline/doctor.go` — diagnósticos
