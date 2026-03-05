---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar ExtractCodeBlocks y ExtractTables

**Story**: [S002 Body Extraction Utilities](README.md)
**Contribuye a**: Utilidades de extraccion de code blocks y tablas disponibles

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

El detector de sub-schema necesita extraer YAML blocks del body para analizar variantes por tipo. El detector de invariantes se beneficia de distinguir prosa de code blocks. goldmark AST tiene `ast.KindFencedCodeBlock` con atributo Language y la extension table para tablas.

## Alcance

**In**:
1. Tipo `CodeBlock struct { Language string; Content string; StartLine int }`
2. `ExtractCodeBlocks(node ast.Node, source []byte) []CodeBlock` — walk FencedCodeBlock nodes
3. Tipo `Table struct { Headers []string; Rows [][]string }`
4. `ExtractTables(node ast.Node, source []byte) []Table` — requiere goldmark extension table
5. Tests para ambas funciones

**Out**: Uso en detectores (F02). Inline code spans (no relevantes para inferencia).

## Estado inicial esperado

- T001 completado (body.go existe con ExtractSections)
- goldmark disponible como dependencia

## Criterios de Aceptacion

- ExtractCodeBlocks extrae bloques con language correcto (go, yaml, empty)
- ExtractCodeBlocks ignora inline code (`backtick`)
- ExtractTables extrae headers y rows correctamente
- Test: documento con 2 code blocks y 1 tabla retorna 2 CodeBlock y 1 Table
- `go test ./internal/extract/ -race` pasa verde

## Fuente de verdad

- `internal/extract/body.go` — archivo creado en T001
- goldmark: `ast.KindFencedCodeBlock`, extension `table`
