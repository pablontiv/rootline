---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Validar links contra schema definido en .stem

**Story**: [S002 Link Schema and Validation](README.md)

## Contexto

Con LinkSchema tipificado (T001) y links extraidos en Records (S001), se necesita validar que los links de cada record cumplan con las restricciones del .stem efectivo: tipo de link esta en allowed, target matchea el glob pattern definido.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: validateLinks (function)
    metodos:
      - nombre: validateLinks
        input: "links []extract.Link, schema LinkSchema, basePath string"
        output: "[]ValidationError"
dependencias_externas: []
tests:
  - Link con tipo en allowed no genera error
  - Link con tipo no en allowed genera error
  - Link con target que matchea glob no genera error
  - Link con target que no matchea glob genera error
  - Sin LinkSchema definida no se validan links (permisivo)
  - Link validation errors incluyen source .stem path
```

## Dependencias

- T001 (LinkSchema)
- S001 (Links en Record)

## Alcance

**In**:
1. Funcion `validateLinks(links, schema, basePath)` retorna []ValidationError
2. Check 1: link.Type esta en schema.Allowed (si Allowed no es nil)
3. Check 2: link.Target matchea schema.Rules[link.Type].Target glob (si regla existe)
4. Integrar en Validate() existente — llamar validateLinks despues de schema/rule checks
5. ValidationError para links incluye: field="links", rule="link_type"/"link_target", source

**Out**: Link target existence check (verificar que archivo existe), broken link detection (eso es graph --check)

## Estado inicial esperado

- LinkSchema tipificado (T001)
- Record.Links poblado (S001)
- Validate() funcional

## Criterios de Aceptacion

- Record con link tipo "blocks" y schema allowed:[blocks,parent] no genera error
- Record con link tipo "unknown" y schema allowed:[blocks,parent] genera error "link type not allowed"
- Record con link target "T003.md" y rule target "*.md" no genera error
- Record con link target "T003.txt" y rule target "*.md" genera error "link target mismatch"
- Sin links schema, links no se validan (exit code 0)
- `go test ./internal/rules/ -race` pasa

## Fuente de verdad

- `internal/rules/validate.go` — Validate function
- `internal/rules/rules.go` — LinkSchema, LinkRule
- `internal/extract/extract.go` — Link, Record.Links
