---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Agregar campo Links al Record y conectar con MarkdownExtractor

**Story**: [S001 Link Extraction](README.md)

## Contexto

Con ParseLinks funcional (T001), se necesita agregar el campo `Links []Link` al Record struct y hacer que MarkdownExtractor lo pueble automaticamente despues de extraer el body. Los links deben aparecer en query results y en describe output.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/extract
interfaces:
  - nombre: Record (extended)
    metodos: []
dependencias_externas: []
tests:
  - Record de archivo con wiki-links tiene Links poblado
  - Record de archivo sin links tiene Links vacio
  - Links aparecen en JSON serialization de Record
  - Query results incluyen links
```

## Dependencias

- T001 (ParseLinks)

## Alcance

**In**:
1. Agregar `Links []Link` al Record struct con json tag
2. MarkdownExtractor llama ParseLinks(body) despues de extraer frontmatter/body
3. Links se serializan en JSON output de todos los comandos
4. --field "links" extrae el array de links
5. Tests que verifican integracion

**Out**: Link indexing, link-based queries, link validation

## Estado inicial esperado

- T001 completado (ParseLinks funcional)
- Record struct en internal/extract/extract.go

## Criterios de Aceptacion

- MarkdownExtractor con archivo que contiene `[[T003]]` produce Record con Links[0].Target == "T003"
- Record.Links serializa correctamente en JSON
- `rootline query --field links` extrae array de links
- Records sin wiki-links tienen Links == nil o []
- Tests existentes de extract siguen pasando
- `go test ./internal/extract/ -race` pasa

## Fuente de verdad

- `internal/extract/extract.go` — Record struct, MarkdownExtractor
- `internal/extract/extract_test.go` — tests existentes
