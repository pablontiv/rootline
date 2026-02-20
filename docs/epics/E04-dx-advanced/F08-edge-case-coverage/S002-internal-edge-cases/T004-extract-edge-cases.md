---
estado: Pending
tipo: software-test
ejecutable_en: 1 sesion
---
# T004: extract_test.go — Unicode, YAML block scalar, sin newline final

**Story**: [S002 Internal Edge Cases](README.md)

## Contexto

El extractor de frontmatter en `internal/extract/` tiene un fallback YAML parser. Los tests existentes cubren BOM, Windows line endings, y YAML malformado. Faltan: valores Unicode en frontmatter (nombres en japones, emojis en titulos), YAML block scalars (`|` literal block y `>` folded), y archivos sin newline final al cierre del frontmatter.

## Alcance

**In**: Agregar a `internal/extract/extract_test.go`:
1. `TestExtract_UnicodeValues` — frontmatter con `titulo: "Instalación de K8s 🚀"` y `autor: "田中"` → extrae correctamente como strings Unicode
2. `TestExtract_YAMLBlockScalar_Literal` — frontmatter con campo usando `|` (literal block scalar multiline) → extrae como string con newlines
3. `TestExtract_YAMLBlockScalar_Folded` — frontmatter con campo usando `>` (folded scalar) → extrae como string
4. `TestExtract_NoTrailingNewline` — archivo que termina en `---` sin `\n` final → extrae frontmatter sin error

**Out**: Cambios al extractor, tests de links

## Estado inicial esperado

- `internal/extract/extract_test.go` existe con tests de BOM, Windows line endings, YAML malformado
- `go test ./internal/extract/ -race` pasa

## Criterios de Aceptacion

- `go test ./internal/extract/ -run TestExtract_Unicode -v` pasa
- `go test ./internal/extract/ -run TestExtract_YAML -v` pasa
- `go test ./internal/extract/ -run TestExtract_NoTrailingNewline -v` pasa
- `go test ./internal/extract/ -race` pasa sin regresiones
- Los valores Unicode se extraen como strings correctos (no como bytes escapados)

## Fuente de verdad

- `internal/extract/extract_test.go` — archivo a extender
- `internal/extract/extract.go` — implementacion del extractor (con fallback YAML parser)
