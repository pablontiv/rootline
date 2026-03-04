---
estado: Specified
tipo: historia
---
# S001: Deterministic Categories 5/7/8/10

**Feature**: [F02 Inference Category Expansion](../README.md)
**Capacidad**: Categorias 100% engine (5, 7, 8, 10) implementadas como detectores Go
**Cubre**: Milestone de F02 — detectores producen inferencias tipadas

## Antes / Despues

**Antes**: Solo categorias 1-4 estan implementadas. Cat 5 tiene LinkSchema struct definida pero sin validacion. Cat 7/8/10 no tienen detector.

**Despues**: 4 detectores nuevos producen inferencias: Cat 5 (link-type validation), Cat 7 (back-reference consistency), Cat 8 (constant detection), Cat 10 (cross-epic path references). Todos son pure Go sin LLM.

## Criterios de Aceptacion (semanticos)

- [ ] Cat 5: LinkSchema.Allowed se valida contra links reales — inferencias de tipo `link_type_violation`
- [ ] Cat 7: Back-references son bidireccionales — inferencias de tipo `missing_back_reference`
- [ ] Cat 8: Fields con 100% mismo valor detectados como constantes — inferencias de tipo `constant_field`
- [ ] Cat 10: Path references (`E\d{2}/F\d{2}`) extraidas y validadas contra filesystem
- [ ] `go test ./... -race` pasa verde

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-cat-5-link-validation.md) | Implementar Cat 5 link-type validation usando LinkSchema |
| [T002](T002-cat-7-back-references.md) | Implementar Cat 7 back-reference consistency check |
| [T003](T003-cat-8-10-constants-crossrefs.md) | Implementar Cat 8 constant detection y Cat 10 cross-epic refs |
| [T004](T004-deterministic-cats-tests.md) | Tests para categorias 5/7/8/10 |

## Fuente de verdad

- `internal/rules/rules.go` — LinkSchema struct con Allowed field
- `internal/graph/graph.go` — Build() para back-references
- `internal/infer/infer.go` — Analyze() con FieldStats
