---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar parser de wiki-links con soporte para tipos

**Story**: [S001 Link Extraction](README.md)

## Contexto

Los documentos Markdown pueden contener wiki-links en formato `[[target]]` (link simple) o `[[type:target]]` (link tipado). El parser debe extraer estos links del body text, ignorando los que estan dentro de code blocks (inline o fenced). Se puede usar regex o integrar goldmark con wikilink extension.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/extract
interfaces:
  - nombre: Link
    metodos: []
  - nombre: ParseLinks
    metodos:
      - nombre: ParseLinks
        input: "body string"
        output: "[]Link"
dependencias_externas:
  - go.abhg.dev/goldmark/wikilink (opcional, puede usar regex)
tests:
  - "[[target]]" retorna Link{Target:"target", Type:"reference"}
  - "[[blocks:T003]]" retorna Link{Target:"T003", Type:"blocks"}
  - "[[parent:../feature]]" retorna Link{Target:"../feature", Type:"parent"}
  - Link en code block se ignora
  - Link en inline code se ignora
  - Multiple links en un documento se extraen todos
  - Texto sin links retorna slice vacio
```

## Dependencias

- Ninguna (puede ser standalone o usar goldmark)

## Alcance

**In**:
1. Struct `Link` con fields: Target string, Type string, Line int
2. Funcion `ParseLinks(body string) []Link`
3. Formato: `[[target]]` → Type="reference", `[[type:target]]` → Type=type
4. Ignorar links en fenced code blocks (``` ``` ```)
5. Ignorar links en inline code (`` ` ``)
6. Reportar numero de linea de cada link
7. Tests unitarios

**Out**: Link resolution (verificar que target existe), link validation contra schema, goldmark integration si regex es suficiente

## Estado inicial esperado

- internal/extract/ con MarkdownExtractor funcional
- Body text disponible en Record.Body

## Criterios de Aceptacion

- `ParseLinks("see [[T003]]")` retorna [{Target:"T003", Type:"reference", Line:1}]
- `ParseLinks("[[blocks:T003]]")` retorna [{Target:"T003", Type:"blocks", Line:1}]
- `ParseLinks("` + "```\n[[ignored]]\n```" + `")` retorna []
- `ParseLinks("no links here")` retorna []
- `ParseLinks("[[a]] and [[b:c]]")` retorna 2 links
- `go test ./internal/extract/ -race` pasa

## Fuente de verdad

- `internal/extract/extract.go` — MarkdownExtractor, Record
