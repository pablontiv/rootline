---
estado: Pending
tipo: historia
cliente: Platform Owner
---
# S001: Graph Field Resolution

**Feature**: [F01 Link Field Resolution](../README.md)
**Capacidad**: DeriveRecord puede acceder a valores de campos de documentos enlazados via wiki-links tipados

## Antes / Despues

**Antes**: DeriveRecord solo accede a campos del propio documento y de children (via aggregate). LinkRule.Field se parsea del .stem pero nunca se usa. No hay forma de referenciar el estado de un documento enlazado en una expresion derive.

**Despues**: DeriveAll construye un RecordResolver (map de path → record). DeriveRecord recibe el resolver, itera links del documento, busca la LinkRule, y si `rule.Field != ""`, inyecta en el env los valores del campo del documento enlazado. Expresiones como `all(blocked_by, {. == 'Completado'})` funcionan.

## Criterios de Aceptacion (semanticos)

- [ ] RecordResolver disponible en DeriveRecord
- [ ] LinkRule.Field consumido para inyectar valores enlazados en env
- [ ] Expresiones derive referencian valores de documentos enlazados correctamente

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-resolve-linked-fields.md) | Wire RecordResolver y LinkRule.Field en derive pipeline |
| [T002](T002-derive-env-linked-values.md) | Tests y documentacion de linked values en derive expressions |

## Fuente de verdad

- `internal/derive/pipeline.go`
- `internal/derive/record.go`
- `internal/rules/rules.go` (LinkRule struct)
