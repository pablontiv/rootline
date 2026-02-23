---
estado: Completado
tipo: feature
---
# F05: Dependency Graph

**Epic**: [E04](../README.md)
**Objetivo**: Rootline extrae, valida, y visualiza relaciones entre documentos via wiki-links tipados con deteccion de ciclos
**Beneficio**: Dependencias entre documentos son explicitas, validadas, y visualizables. Estado derivado por propagacion (feature bloqueada si task pendiente).
**Milestone**: `[[blocks:T003]]` en documento se extrae como link tipado. `.stem` define links permitidos. `rootline graph docs/` genera diagrama. `rootline graph --check` detecta ciclos.

## Scope

**In**: Wiki-link extraction, link types en Record, link schema en .stem, link validation, graph builder, cycle detection, graph command con DOT/mermaid output
**Out**: Bidirectional link resolution, link-based queries, graph visualization UI, link refactoring tools

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Link Extraction](S001-link-extraction/) | MarkdownExtractor extrae wiki-links tipados del body |
| S002 | [Link Schema and Validation](S002-link-schema-validation/) | .stem define links permitidos y targets validos |
| S003 | [Graph Command](S003-graph-command/) | Visualizar dependencias y detectar ciclos |
| S004 | [Graph Schema-Aware Link Filtering](S004-graph-schema-filtering/) | graph --check solo evalua links que .stem define como estructurales |

## Dependencias

- Links field ya parsea en StemFile (internal/rules/rules.go)
- Estado derivado por propagacion (Level 3 derivation) requiere F04, diferido

## Fuente de verdad

- `internal/rules/rules.go` — StemFile.Links (map[string]any, reservado)
- `internal/extract/extract.go` — Record struct (agregar Links field)
- `docs/research/I9-opportunity-areas.md` — seccion 6.4 Graph de Dependencias
