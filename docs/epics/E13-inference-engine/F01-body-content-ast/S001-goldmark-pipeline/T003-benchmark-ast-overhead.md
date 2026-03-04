---
estado: Completed
tipo: test
ejecutable_en: 1 sesion
---
# T003: Benchmark AST overhead y feature flag

**Story**: [S001 goldmark Pipeline Integration](README.md)
**Contribuye a**: Overhead ≤20% verificado; feature flag para opt-in

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`

## Contexto

goldmark parsing añade overhead al pipeline de extraccion. La investigacion estima ≤20% como aceptable. Se necesita un benchmark para verificar y un mecanismo para opt-in/opt-out del AST parsing.

## Alcance

**In**:
1. Benchmark `BenchmarkExtractWithAST` vs `BenchmarkExtractWithoutAST` en extract_test.go
2. Medir overhead en archivos pequeños (~50 lineas) y medianos (~500 lineas)
3. Si overhead >20%, añadir flag `parseAST bool` en MarkdownExtractor para opt-in
4. Documentar resultado del benchmark en el commit message

**Out**: Optimizacion de goldmark (si overhead es aceptable, no optimizar). Decidir si reemplazar ParseLinks por ParseLinksAST (decision futura).

## Estado inicial esperado

- T001 y T002 completados
- AST se parsea en Extract()

## Criterios de Aceptacion

- `go test ./internal/extract/ -bench BenchmarkExtract -benchmem` produce resultados comparables
- Overhead documentado: ns/op con AST vs sin AST
- Si overhead >20%: flag `parseAST` existe y AST no se parsea por defecto
- Si overhead ≤20%: AST se parsea siempre (no necesita flag)
- `go test ./... -race` pasa verde

## Fuente de verdad

- `internal/extract/extract.go` — MarkdownExtractor.Extract()
- `internal/extract/extract_test.go` — benchmarks
