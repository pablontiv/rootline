---
estado: Specified
tipo: historia
---
# S003: Semantic Category Engine Stubs 9/11

**Feature**: [F02 Inference Category Expansion](../README.md)
**Capacidad**: Porciones engine de categorias con alto % LLM (9: deps heterogeneas, 11: traceability)
**Cubre**: Milestone de F02 — stubs engine extraen datos para futuro agent

## Antes / Despues

**Antes**: Cat 9 (heterogeneous dependencies) y Cat 11 (traceability) no estan implementadas. El engine no extrae datos formales de dependencias ni links de traceability.

**Despues**: Cat 9 engine extrae dependencias formales (wiki-links, path references, tabla de dependencias) sin disambiguation semantica. Cat 11 engine extrae links "Contribuye a" y "Cubre" sin semantic matching. Ambos producen datos parciales que un futuro agent completara.

## Criterios de Aceptacion (semanticos)

- [ ] Cat 9: Extrae dependencias formales de links + secciones "Dependencias" del body
- [ ] Cat 11: Extrae claims de traceability (`Contribuye a`, `Cubre`, `Satisface`) del body
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
| [T001](T001-cat-9-formal-deps.md) | Implementar Cat 9 engine portion — formal dependency extraction |
| [T002](T002-cat-11-traceability-links.md) | Implementar Cat 11 engine portion — traceability link extraction |
| [T003](T003-semantic-stubs-tests.md) | Tests para porciones engine de cat 9/11 |

## Fuente de verdad

- `internal/extract/links.go` — ParseLinks para wiki-links formales
- `internal/extract/body.go` — ExtractSections para localizar secciones de dependencias
- Proporciones: Cat 9 = 30% Go / 70% LLM, Cat 11 = 20% Go / 80% LLM (Apendice A)
