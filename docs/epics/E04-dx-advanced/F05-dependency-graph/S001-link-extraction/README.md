# S001: Link Extraction

**Feature**: [F05 Dependency Graph](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: MarkdownExtractor detecta y extrae wiki-links tipados del body de documentos Markdown, poblando el campo Links del Record

## Antes / Despues

**Antes**: Wiki-links como `[[T003]]` o `[[blocks:T003]]` en el body de documentos Markdown son texto opaco. No se extraen, no se indexan, no se validan. Rootline no sabe que relaciones existen entre documentos.

**Despues**: MarkdownExtractor parsea el body y extrae links con tipo y target. `Record.Links` contiene una lista de `Link{Target, Type, Line}`. Links sin tipo explicito usan type "reference". Links dentro de code blocks se ignoran. Links aparecen en query results y describe output.

## Criterios de Aceptacion (semanticos)

- [ ] `[[target]]` se extrae como Link con tipo "reference"
- [ ] `[[blocks:target]]` se extrae como Link con tipo "blocks"
- [ ] Links dentro de code blocks (``` ```) no se extraen
- [ ] Record.Links esta disponible en query results

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-wikilink-parser.md) | Parser de wiki-links con soporte para tipos |
| [T002](T002-record-links-field.md) | Agregar campo Links al Record y conectar con extractor |

## Fuente de verdad

- `internal/extract/extract.go` — Record, MarkdownExtractor
