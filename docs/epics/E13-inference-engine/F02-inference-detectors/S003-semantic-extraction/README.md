---
estado: Specified
tipo: historia
---
# S003: Semantic Category Engine Stubs 9/11

**Feature**: [F02 Inference Category Expansion](../README.md)
**Capacidad**: Porciones engine de categorias con alto % LLM (9: deps heterogeneas, 11: traceability)
**Cubre**: Milestone de F02 — stubs engine extraen datos para futuro agent

## Antes / Despues

**Antes**: Formal dependency extraction y traceability link extraction no estan implementadas. El engine no extrae datos formales de dependencias ni links de traceability.

**Despues**: Formal dependency extraction extrae dependencias formales (wiki-links, path references, tabla de dependencias) sin disambiguation semantica. Traceability link extraction extrae links "Contribuye a" y "Cubre" sin semantic matching. Ambos producen datos parciales que un futuro agent completara.

## Criterios de Aceptacion (semanticos)

- [ ] Formal dependency extraction: Extrae dependencias formales de links + secciones "Dependencias" del body
- [ ] Traceability link extraction: Extrae claims de traceability (`Contribuye a`, `Cubre`, `Satisface`) del body
- [ ] Ambos producen inferencias tipadas con flag `requires_agent: true` para porciones LLM
- [ ] `go test ./... -race` pasa verde

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-formal-dependency-stubs.md) | Implementar formal dependency extraction (engine portion) |
| [T002](T002-traceability-link-stubs.md) | Implementar traceability link extraction (engine portion) |
| [T003](T003-semantic-extraction-tests.md) | Tests para porciones engine de formal deps / traceability |

## Fuente de verdad

- `internal/extract/links.go` — ParseLinks para wiki-links formales
- `internal/extract/body.go` — ExtractSections para localizar secciones de dependencias
- Proporciones: formal deps = 30% Go / 70% LLM, traceability = 20% Go / 80% LLM (Apendice A)
