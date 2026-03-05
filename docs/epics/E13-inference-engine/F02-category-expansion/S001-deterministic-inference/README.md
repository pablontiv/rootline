---
estado: Specified
tipo: historia
---
# S001: Deterministic Inference

**Feature**: [F02 Inference Category Expansion](../README.md)
**Capacidad**: Categorias 100% engine (5, 7, 8, 10) implementadas como detectores Go
**Cubre**: Milestone de F02 — detectores producen inferencias tipadas

## Antes / Despues

**Antes**: Solo categorias 1-4 estan implementadas. Link-type validation tiene LinkSchema struct definida pero sin validacion. Back-reference/constant/cross-reference detectors no existen.

**Despues**: 4 detectores nuevos producen inferencias: link-type validation, back-reference consistency, constant field detection, cross-reference validation. Todos son pure Go sin LLM.

## Criterios de Aceptacion (semanticos)

- [ ] Link-type validation: LinkSchema.Allowed se valida contra links reales — inferencias de tipo `link_type_violation`
- [ ] Back-reference consistency: Back-references son bidireccionales — inferencias de tipo `missing_back_reference`
- [ ] Constant field detection: Fields con 100% mismo valor detectados como constantes — inferencias de tipo `constant_field`
- [ ] Cross-reference validation: Path references (`E\d{2}/F\d{2}`) extraidas y validadas contra filesystem
- [ ] `go test ./... -race` pasa verde

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-link-type-validation.md) | Implementar link-type validation usando LinkSchema |
| [T002](T002-back-reference-consistency.md) | Implementar back-reference consistency check |
| [T003](T003-constant-detection-crossrefs.md) | Implementar constant field detection y cross-reference validation |
| [T004](T004-deterministic-inference-tests.md) | Tests de integracion para detectores deterministicos |

## Fuente de verdad

- `internal/rules/rules.go` — LinkSchema struct con Allowed field
- `internal/graph/graph.go` — Build() para back-references
- `internal/infer/infer.go` — Analyze() con FieldStats
