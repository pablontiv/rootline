---
estado: Specified
tipo: test
ejecutable_en: 1 sesion
---
# T003: Tests con edge cases de extraccion

**Story**: [S002 Body Extraction Utilities](README.md)
**Contribuye a**: Edge cases de extraccion cubiertos

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Las utilidades de extraccion (ExtractSections, ExtractCodeBlocks, ExtractTables) necesitan manejar edge cases reales de documentos rootline: headings en code blocks, tablas malformadas, documentos sin frontmatter (body = todo), markdown con HTML embebido.

## Alcance

**In**:
1. Test: heading `##` sin espacio (no es heading valido segun CommonMark)
2. Test: heading dentro de blockquote (`> ## Heading`)
3. Test: tabla sin separador `|---|` (no es tabla valida)
4. Test: code block sin language tag
5. Test: documento vacio (body vacio despues de frontmatter)
6. Test: nested fenced code blocks (triple backtick dentro de cuadruple backtick)
7. Corregir bugs encontrados en T001/T002 si los hay

**Out**: Tests de integracion con categorias de inferencia (F02).

## Estado inicial esperado

- T001 y T002 completados
- ExtractSections, ExtractCodeBlocks, ExtractTables implementados

## Criterios de Aceptacion

- ≥6 edge case tests implementados
- Todos pasan: `go test ./internal/extract/ -race -run TestExtract`
- Coverage de body.go ≥90% (edge cases cubren branches)
- No hay panics en ningun edge case (funciones retornan slices vacios, no nil)

## Fuente de verdad

- `internal/extract/body.go` — funciones bajo test
- `internal/extract/body_test.go` — tests
- CommonMark spec para validez de headings y tablas
