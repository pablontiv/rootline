# S002: Markdown Extraction

**Feature**: [F02 Core Engine](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Un unico MarkdownExtractor produce Records JSON estructurados desde cualquier archivo .md, reemplazando 7 parsers independientes

## Antes / Despues

**Antes**: 7 sistemas de parsing independientes con 4 patrones regex distintos para el mismo campo `estado:`. Python `line.split(':', 1)`, bash `grep -q`, LLM prompts. Failures silenciosos cuando el formato cambia.

**Despues**: `MarkdownExtractor` implementa la interfaz `Extractor` (I7). Produce `Record{Path, Type, Frontmatter map[string]any, Body, Errors}`. YAML frontmatter via `gopkg.in/yaml.v3` con fallback para frontmatter malformado. Un solo punto de parsing, testeable, determinista.

## Criterios de Aceptacion (semanticos)

- [ ] Archivos .md con frontmatter YAML producen Records con Frontmatter poblado
- [ ] Archivos .md sin frontmatter producen Records con Body completo y Frontmatter vacio
- [ ] Frontmatter malformado produce Record parcial con ExtractionErrors (no fallo total)

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-markdown-extractor.md) | Implementar MarkdownExtractor con Record type |
| [T002](T002-extractor-registry.md) | Implementar Registry con lookup por extension |

## Fuente de verdad

- `src/rootline/docs/research/I7-extractors-architecture.md` — Extractor interface, Record type, edge cases
