---
estado: Specified
tipo: historia
---
# S002: Body-Aware Categories 6/12/13

**Feature**: [F02 Inference Category Expansion](../README.md)
**Capacidad**: Categorias que usan goldmark AST para extraccion estructural de body content
**Cubre**: Milestone de F02 — detectores body-aware producen inferencias tipadas

## Antes / Despues

**Antes**: No hay analisis estructural de body content. Cat 6 (body sections), Cat 12 (invariants), Cat 13 (sub-schema by type) no estan implementadas. El body se trata como string plano.

**Despues**: Cat 6 detecta patrones de estructura de secciones. Cat 12 extrae invariantes (`INV\d+:` regex + AST section boundaries). Cat 13 detecta sub-schemas por tipo usando code blocks YAML y FieldStats por subgrupo. Todos usan goldmark AST de F01.

## Criterios de Aceptacion (semanticos)

- [ ] Cat 6: Detecta patrones de heading structure (secciones requeridas/opcionales)
- [ ] Cat 12: Extrae invariantes con regex `INV\d+:` dentro de secciones especificas
- [ ] Cat 13: Agrupa records por tipo y detecta fields exclusivos por tipo
- [ ] Todas las extracciones usan AST, no regex sobre body plano
- [ ] `go test ./... -race` pasa verde

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-cat-6-body-sections.md) | Implementar Cat 6 body section structure analysis |
| [T002](T002-cat-12-invariant-extraction.md) | Implementar Cat 12 invariant extraction via regex + AST |
| [T003](T003-cat-13-sub-schema-detection.md) | Implementar Cat 13 sub-schema detection per type group |
| [T004](T004-body-aware-cats-tests.md) | Tests para categorias 6/12/13 |

## Fuente de verdad

- `internal/extract/body.go` — ExtractSections, ExtractCodeBlocks (de F01/S002)
- `internal/infer/infer.go` — Analyze(), FieldStats
- Categorias definidas en intake/inference-engine-architecture.md Apendice A
