---
estado: Specified
tipo: feature
---
# F01: Body Content AST Infrastructure

**Epic**: [E13 Inference Engine](../README.md)
**Satisface**: P1
**Objetivo**: goldmark integrado en el pipeline de extraccion con utilidades para analisis estructural de body content
**Beneficio**: Habilita categorias body-aware (6/12/13) y mejora precision de extraccion de links
**Milestone**: Record tiene AST opcional; ExtractSections y ExtractCodeBlocks disponibles como utilidades

## Scope

**In**: Dependencia goldmark, campo AST en Record, ParseLinksAST, utilidades de extraccion (sections, code blocks, tables)
**Out**: Implementacion de categorias de inferencia (eso es F02)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [goldmark Pipeline Integration](S001-goldmark-pipeline/) | goldmark parsea body a AST sin romper contratos existentes |
| S002 | [Body Extraction Utilities](S002-body-extraction-utils/) | Utilidades para extraer secciones, code blocks y tablas del AST |

## Invariantes

- INV1 (heredado): `go test ./... -race` pasa verde en cada commit
- INV2 (heredado): Contratos JSON mantienen `"version": 1`
- INV3 (heredado): Coverage ≥85%

## Dependencias

- Ninguna (este Feature es foundation)

## Fuente de verdad

- `internal/extract/extract.go` — Record struct, MarkdownExtractor
- `internal/extract/links.go` — ParseLinks (sera complementado con ParseLinksAST)
- `go.mod` — dependencias actuales (goldmark se añade aqui)
