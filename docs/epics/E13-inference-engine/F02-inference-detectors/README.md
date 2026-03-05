---
estado: Specified
tipo: feature
---
# F02: Inference Category Expansion

**Epic**: [E13 Inference Engine](../README.md)
**Satisface**: P1
**Objetivo**: Implementar las porciones Go de categorias 5-13
**Beneficio**: Engine cubre ~80% de toda inferencia, dejando solo residuo semantico para agent
**Milestone**: 9 categorias nuevas implementadas (5-13), cada una con detector que produce inferencias tipadas

## Scope

**In**: Detectores Go para categorias 5/6/7/8/9/10/11/12/13 (solo porciones engine)
**Out**: Porciones LLM de categorias 9/11 (agent Epic), comando analyze (F03)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Deterministic Inference](S001-structural-detectors/) | Categorias 100% engine implementadas como detectores Go |
| S002 | [Body-Aware Inference](S002-body-aware-inference/) | Categorias que usan goldmark AST para extraccion estructural |
| S003 | [Semantic Category Stubs 9/11](S003-semantic-extraction/) | Porciones engine de cats con alto % LLM |

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
- `internal/rules/structural.go` — ValidateDirectory (cat 4 reference)
- `internal/graph/graph.go` — Build() para back-references
- `internal/extract/links.go` — ParseLinks
