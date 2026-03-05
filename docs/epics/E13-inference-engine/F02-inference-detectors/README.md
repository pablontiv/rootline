---
estado: Specified
tipo: feature
---
# F02: Inference Detectors

**Epic**: [E13 Inference Engine](../README.md)
**Satisface**: P1
**Objetivo**: Implementar 9 detectores nuevos de inferencia en Go
**Beneficio**: Engine cubre la mayoria de inferencias, dejando solo análisis semántico pendiente para agent
**Milestone**: 9 detectores nuevos implementados, cada uno produce inferencias tipadas

## Scope

**In**: Detectores Go para link-type, body-section, back-reference, constant-field, formal-dependency, cross-reference, traceability, invariant, sub-schema
**Out**: Análisis semántico de dependency/traceability (agent Epic), comando analyze (F03)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Structural Detectors](S001-structural-detectors/) | Detectores puramente estructurales: link-type, back-reference, constant-field, cross-reference |
| S002 | [Body-Aware Inference](S002-body-aware-inference/) | Detectores que usan goldmark AST: body-section, invariant, sub-schema |
| S003 | [Semantic Extraction](S003-semantic-extraction/) | Detectores con datos parciales: formal-dependency, traceability |

## Invariantes

- INV1 (heredado): `go test ./... -race` pasa verde en cada commit
- INV2 (heredado): Contratos JSON mantienen `"version": 1`
- INV3 (heredado): Coverage ≥85%

## Dependencias

- S002 y S003 dependen de F01 (goldmark AST)
- S001 es independiente de F01

## Fuente de verdad

- `internal/infer/infer.go` — Analyze(), FieldStats
- `internal/rules/rules.go` — LinkSchema
- `internal/rules/structural.go` — ValidateDirectory (directory structure reference)
- `internal/graph/graph.go` — Build() para back-references
- `internal/extract/links.go` — ParseLinks
