---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar formal dependency extraction

**Story**: [S003 Semantic Extraction](README.md)
**Contribuye a**: Formal dependency detector extrae dependencias formales sin disambiguation semantica

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

El detector de formal dependency extrae dependencias formales: wiki-links `[[blocks:X]]`, path references, y secciones "Dependencias" del body. La disambiguation semantica (¿es esta dependencia real o solo una mencion?) queda para un futuro agent.

## Alcance

**In**:
1. Detector que recibe Record con AST y links
2. Extrae wiki-links con prefijo semantico (`blocks:`, `relates:`, `extends:`)
3. Localiza seccion "Dependencias" via ExtractSections y extrae items de lista
4. Clasifica dependencias: formal (wiki-link) vs informal (mencion en prosa)
5. Produce inferencias de tipo `formal_dependency` y `informal_dependency_candidate` (requiere agent)
6. Flag `requires_agent: true` en inferencias informales

**Out**: Disambiguation semantica de dependencias informales (agent Epic). Crear dependencias automaticamente.

## Estado inicial esperado

- F01 completado (ExtractSections disponible)
- ParseLinks extrae wiki-links con targets

## Criterios de Aceptacion

- Wiki-link `[[blocks:T002]]` → produce `formal_dependency{type: "blocks", target: "T002"}`
- Seccion `## Dependencias\n- Requiere F01 completado` → produce `informal_dependency_candidate{text: "Requiere F01 completado", requires_agent: true}`
- Record sin dependencias → retorna []Inference vacio
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/extract/links.go` — ParseLinks, Link struct
- `internal/extract/body.go` — ExtractSections
- Documentos con dependencias: task files en docs/epics/ (wiki-links blocks:)
