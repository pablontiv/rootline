---
tipo: feature
---
# F01: Link Field Resolution

**Epic**: [E06](../README.md)
**Objetivo**: El derive engine resuelve valores de campos de documentos enlazados via wiki-links, habilitando state propagation entre documentos conectados
**Beneficio**: Tasks bloqueadas se detectan automaticamente. `rootline query` retorna solo tasks accionables sin intervencion manual.
**Milestone**: Derive expression `all(blocked_by, {. == 'Completado'})` evalua correctamente usando valores de documentos enlazados via `[[blocks:]]`

## Scope

**In**: Wire LinkRule.Field, RecordResolver en derive pipeline, dependency state propagation, e2e tests
**Out**: Bidirectional link resolution, link-based queries, graph visualization changes, UI

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Graph Field Resolution](S001-graph-field-resolution/) | DeriveRecord accede a campos de documentos enlazados |
| S002 | [Dependency State Propagation](S002-dependency-state-propagation/) | .stem derive expressions calculan estado basado en dependencias |

## Dependencias

- internal/derive/ (F04 derivation engine — completado)
- internal/graph/ (F05 dependency graph — completado)
- internal/rules/ LinkRule.Field (ya parseado, nunca consumido)

## Fuente de verdad

- `internal/derive/pipeline.go` — DeriveAll pipeline
- `internal/derive/record.go` — DeriveRecord
- `internal/rules/rules.go` — LinkRule.Field definition
- `internal/graph/graph.go` — link resolution logic
