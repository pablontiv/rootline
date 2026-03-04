---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Añadir goldmark y campo AST a Record

**Story**: [S001 goldmark Pipeline Integration](README.md)
**Contribuye a**: Record tiene AST opcional sin romper contratos JSON

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV2: Contratos JSON no cambian
  - Verificar: Confirmar tag `json:"-"` en campo AST

## Contexto

Record struct en `internal/extract/extract.go` (lineas 26-34) tiene Body como string. goldmark (yuin/goldmark) es un parser AST de Markdown pure Go sin dependencias transitivas. El campo AST debe ser opcional (no romper extraccion existente) y no serializarse a JSON.

## Alcance

**In**:
1. `go get github.com/yuin/goldmark` — añadir dependencia
2. Añadir campo `AST ast.Node` con tag `json:"-"` a Record struct
3. Parsear body a AST en MarkdownExtractor.Extract() cuando body no esta vacio
4. `go build ./...` compila sin errores

**Out**: Usar AST para extraccion de links (eso es T002). Utilidades de extraccion (S002).

## Estado inicial esperado

- go.mod tiene 5 dependencias directas
- Record struct tiene Body string sin AST
- `go test ./... -race` pasa verde

## Criterios de Aceptacion

- `go get github.com/yuin/goldmark` no produce conflictos de dependencia
- `grep 'AST.*json:"-"' internal/extract/extract.go` retorna 1 match
- `go build ./...` compila sin errores
- `go test ./... -race` pasa verde (AST no afecta tests existentes)

## Fuente de verdad

- `internal/extract/extract.go` — Record struct, MarkdownExtractor.Extract()
- `go.mod` — dependencias
