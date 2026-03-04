---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar ExtractSections con AST heading walk

**Story**: [S002 Body Extraction Utilities](README.md)
**Contribuye a**: ExtractSections retorna secciones por heading level

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Varias categorias de inferencia (6: body sections, 12: invariants, 13: sub-schema by type) necesitan extraer secciones del body por heading. Actualmente fix.go:236 usa `strings.Index("\n## ")` que es fragil (confunde headings en code blocks). goldmark AST resuelve esto con `ast.KindHeading`.

## Alcance

**In**:
1. Crear `internal/extract/body.go` (o sections.go)
2. Tipo `Section struct { Heading string; Level int; Content string; StartLine int }`
3. `ExtractSections(node ast.Node, source []byte) []Section` — walk AST, split por headings
4. Tests con markdown real: headings normales, headings en code blocks, headings anidados

**Out**: ExtractCodeBlocks y ExtractTables (T002). Uso en categorias de inferencia (F02).

## Estado inicial esperado

- F01/S001 completado (Record tiene AST)
- No existe archivo body.go en internal/extract/

## Criterios de Aceptacion

- `ExtractSections` compila y retorna []Section
- Test: markdown con `## A`, `### B`, `## C` retorna 3 secciones con levels correctos
- Test: heading `## Foo` dentro de fenced code block NO se extrae como seccion
- Test: documento sin headings retorna 1 seccion (todo el body)
- Coverage de body.go ≥85%

## Fuente de verdad

- goldmark: `ast.KindHeading`, `ast.KindFencedCodeBlock`
- `internal/fix/fix.go:236` — ejemplo de lo que se reemplaza
