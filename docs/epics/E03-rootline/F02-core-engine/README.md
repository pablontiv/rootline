# F02: Core Engine

**Epic**: [E03](../README.md)
**Objetivo**: Rootline puede cargar archivos .stem, mergearlos con herencia parent-to-child, y extraer Records desde Markdown
**Beneficio**: Establece el pipeline fundacional que todos los comandos consumen
**Milestone**: Tests unitarios pasan para .stem merge + Markdown extraction produciendo Records correctos

## Scope

**In**: .stem YAML parser, walk-up discovery, type-driven merge, Markdown extractor, extractor registry, directory scanner, scope matching
**Out**: Validation rules, query engine, CLI commands, MCP server

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Stem Parser and Merge](S001-stem-parser-merge/) | Engine carga .stem YAML, camina hasta .git, mergea top-down con reglas type-driven |
| S002 | [Markdown Extraction](S002-markdown-extraction/) | MarkdownExtractor produce Records JSON desde cualquier archivo .md |
| S003 | [File Scanner](S003-file-scanner/) | Scanner recorre directorios, respeta .gitignore, delega a extractors |

## Dependencias

- F01 completado (Go module buildable con cobra skeleton)

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` — arquitectura y pipeline
- `src/rootline/docs/research/I5-describe-contract.md` — merge algorithm
- `src/rootline/docs/research/I7-extractors-architecture.md` — extractor interface y Record type
