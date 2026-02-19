---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar Registry con lookup por extension

**Story**: [S002 Markdown Extraction](README.md)

## Contexto

El Registry mapea extensiones de archivo y nombres a Extractors. `NewRegistry()` registra MarkdownExtractor por defecto. `ForFile(path, stemExtractor)` resuelve cual extractor usar para un archivo dado, con override opcional desde .stem.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/extract
interfaces:
  - nombre: Registry
    metodos:
      - nombre: Register
        input: "e Extractor"
        output: (none, panics on duplicate)
      - nombre: ForFile
        input: "path string, stemExtractor string"
        output: Extractor
dependencias_externas: []
tests:
  - NewRegistry incluye MarkdownExtractor
  - ForFile con .md retorna MarkdownExtractor
  - ForFile con .txt retorna nil
  - ForFile con stemExtractor override usa nombre
  - Register con extension duplicada panic
```

## Dependencias

- T001 completado (Extractor interface y MarkdownExtractor disponibles)

## Alcance

**In**:
1. Struct `Registry` con maps byName y byExtension
2. `NewRegistry()` crea registry con MarkdownExtractor registrado
3. `Register(e Extractor)` agrega extractor (panic on duplicate)
4. `ForFile(path, stemExtractor)` resuelve extractor

**Out**: Plugin extractors (I2 deferred), scanner integration

## Estado inicial esperado

- T001 completado: Extractor interface y MarkdownExtractor en internal/extract/

## Criterios de Aceptacion

- `NewRegistry().ForFile("doc.md", "")` retorna MarkdownExtractor
- `NewRegistry().ForFile("data.json", "")` retorna nil
- `NewRegistry().ForFile("doc.md", "markdown")` retorna MarkdownExtractor (by name)
- `Register` con extension duplicada hace panic
- `Register` con nombre duplicado hace panic

## Fuente de verdad

- `src/rootline/docs/research/I7-extractors-architecture.md` seccion 4.3 (The Registry)
