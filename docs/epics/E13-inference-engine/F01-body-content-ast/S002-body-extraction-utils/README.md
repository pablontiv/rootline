---
estado: Completed
tipo: historia
---
# S002: Body Extraction Utilities

**Feature**: [F01 Body Content AST Infrastructure](../README.md)
**Capacidad**: Utilidades para extraer secciones, code blocks y tablas del AST de goldmark
**Cubre**: Milestone de F01 — ExtractSections y ExtractCodeBlocks disponibles

## Antes / Despues

**Antes**: No hay forma de extraer secciones estructurales del body. Regex naive (`\n## `) es fragil para headings dentro de code blocks o listas.

**Despues**: `ExtractSections(ast)` retorna secciones por heading level. `ExtractCodeBlocks(ast)` retorna bloques con lenguaje y contenido. `ExtractTables(ast)` retorna tablas parseadas. Todas usan AST walk, no regex.

## Criterios de Aceptacion (semanticos)

- [ ] ExtractSections retorna slice de secciones con heading text, level, y body content
- [ ] ExtractCodeBlocks retorna slice con language y content por bloque
- [ ] ExtractTables retorna tablas como slices de headers + rows
- [ ] Edge cases: headings en code blocks no se confunden con secciones reales
- [ ] `go test ./... -race` pasa verde

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-extract-sections.md) | Implementar ExtractSections con AST heading walk |
| [T002](T002-extract-codeblocks-tables.md) | Implementar ExtractCodeBlocks y ExtractTables |
| [T003](T003-extraction-edge-case-tests.md) | Tests con edge cases de extraccion |

## Fuente de verdad

- `internal/extract/` — nuevo archivo body.go (o sections.go)
- goldmark AST API: `ast.KindHeading`, `ast.KindFencedCodeBlock`, extension table
