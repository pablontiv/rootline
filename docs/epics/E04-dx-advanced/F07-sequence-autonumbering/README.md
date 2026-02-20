---
estado: Pending
tipo: feature
---
# F07: Sequence Auto-numbering

**Epic**: [E04](../README.md)
**Objetivo**: El tipo `sequence` en `.stem` permite que `rootline describe <dir> --field schema.id.next` retorne el proximo ID disponible basado en archivos existentes
**Beneficio**: Elimina los bash one-liners de auto-numbering en los skills; el engine rootline es la fuente de verdad para secuencias de IDs (T001, S001, F01, E01)
**Milestone**: `rootline describe docs/epics/E04-dx-advanced/F07-sequence-autonumbering/S001-core-engine/ --field schema.id.next` retorna "T004"

## Scope

**In**: SchemaField con Prefix/Digits/Next, computeNextSequence en describe.go, .stem files de planning con type: sequence, tests unitarios y CLI
**Out**: Auto-increment persistente en base de datos, sequence en validation rules, cambios al merge behavior de sequence fields

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-core-engine/) | Core Engine | SchemaField.Next se computa y aparece en JSON de describe |
| [S002](S002-schema-wiring/) | Schema Wiring | .stem files del arbol de planificacion definen sus secuencias |

## Dependencias

- Ninguna feature previa requerida
- S002 depende de S001

## Fuente de verdad

- `internal/rules/rules.go` — SchemaField struct
- `internal/rules/describe.go` — NewDescribeResult y computeNextSequence (nuevo)
- `docs/epics/.stem` — schema base a extender
