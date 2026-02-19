---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar MarkdownExtractor con Record type

**Story**: [S002 Markdown Extraction](README.md)

## Contexto

El Extractor interface (I7) define 3 metodos: Extract, Extensions, Name. MarkdownExtractor es el unico extractor built-in. Recibe path y content como bytes, retorna un Record con frontmatter parseado y body separado. Incluye fallback parser para frontmatter malformado (resilience sobre correctness).

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/extract
interfaces:
  - nombre: Extractor
    metodos:
      - nombre: Extract
        input: "path string, content []byte"
        output: "*Record, error"
      - nombre: Extensions
        input: (none)
        output: "[]string"
      - nombre: Name
        input: (none)
        output: string
dependencias_externas:
  - gopkg.in/yaml.v3
tests:
  - Frontmatter YAML valido + body
  - Sin frontmatter (todo es body)
  - Frontmatter malformado (fallback parser)
  - Archivo vacio (0 bytes)
  - BOM al inicio del archivo
  - Frontmatter sin body (metadata-only)
  - Delimiter no cerrado
  - Keys YAML duplicadas (last wins)
```

## Dependencias

- F01/S001 completado (paquete internal/extract/ existe)

## Alcance

**In**:
1. Definir interface `Extractor` (Extract, Extensions, Name)
2. Definir struct `Record` (Path, Type, Frontmatter map[string]any, Body string, Errors []ExtractionError)
3. Definir struct `ExtractionError` (Line int, Message string)
4. Implementar `MarkdownExtractor` con:
   - BOM stripping
   - Frontmatter detection (`---\n`)
   - YAML parsing via `gopkg.in/yaml.v3`
   - Fallback line-by-line parser para YAML malformado
   - Body separation
5. Tests para los 8 edge cases de I7 (EC-1 through EC-8)

**Out**: Registry (T002), file I/O (scanner), validation

## Estado inicial esperado

- Paquete `internal/extract/` existe y compila

## Criterios de Aceptacion

- `Extract("test.md", validContent)` retorna Record con Frontmatter y Body correctos
- `Extract("test.md", noFrontmatter)` retorna Record con Frontmatter vacio y Body completo
- `Extract("test.md", malformedYAML)` retorna Record parcial con Errors no vacio
- `Extract("test.md", emptyFile)` retorna Record vacio sin error fatal
- `Extract("test.md", bomContent)` strip BOM y parsea correctamente
- `Extensions()` retorna `[".md", ".markdown"]`
- `Name()` retorna `"markdown"`
- Tests cubren los 8 edge cases documentados en I7

## Fuente de verdad

- `src/rootline/docs/research/I7-extractors-architecture.md` seccion 4 (Core Specification) y 5 (Practical Application)
- `src/rootline/docs/research/I7-extractors-architecture.md` seccion 7 (Edge Cases)
