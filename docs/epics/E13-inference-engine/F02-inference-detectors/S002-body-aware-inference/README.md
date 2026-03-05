---
estado: Specified
tipo: historia
---
# S002: Body-Aware Inference

**Feature**: [F02 Inference Category Expansion](../README.md)
**Capacidad**: Categorias que usan goldmark AST para extraccion estructural de body content
**Cubre**: Milestone de F02 — detectores body-aware producen inferencias tipadas

## Antes / Despues

**Antes**: No hay analisis estructural de body content. Body section analysis, invariant extraction, y sub-schema detection no estan implementadas. El body se trata como string plano.

**Despues**: Body section analysis detecta patrones de estructura de secciones. Invariant extraction extrae invariantes (`INV\d+:` regex + AST section boundaries). Sub-schema detection detecta sub-schemas por tipo usando code blocks YAML y FieldStats por subgrupo. Todos usan goldmark AST de F01.

## Criterios de Aceptacion (semanticos)

- [ ] Body section analysis: Detecta patrones de heading structure (secciones requeridas/opcionales)
- [ ] Invariant extraction: Extrae invariantes con regex `INV\d+:` dentro de secciones especificas
- [ ] Sub-schema detection: Agrupa records por tipo y detecta fields exclusivos por tipo
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
| [T001](T001-body-section-analysis.md) | Implementar body section structure analysis |
| [T002](T002-invariant-extraction.md) | Implementar invariant extraction via regex + AST |
| [T003](T003-subschema-detection.md) | Implementar sub-schema detection per type group |
| [T004](T004-body-analysis-tests.md) | Tests de integracion para detectores body-aware |

## Fuente de verdad

- `internal/extract/body.go` — ExtractSections, ExtractCodeBlocks (de F01/S002)
- `internal/infer/infer.go` — Analyze(), FieldStats
- Categorias definidas en intake/inference-engine-architecture.md Apendice A
