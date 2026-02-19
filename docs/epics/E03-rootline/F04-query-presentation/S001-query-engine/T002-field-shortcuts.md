---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Resolver shortcuts de campos comunes

**Story**: [S001 Query Engine](README.md)

## Contexto

Los usuarios escriben `estado eq Pending` en vez de `frontmatter.estado eq Pending`. Los shortcuts son azucar sintactico que resuelve nombres cortos a dot-paths completos. Los shortcuts por defecto cubren los campos mas usados. Shortcuts adicionales se definen per .stem scope.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/query
interfaces:
  - nombre: ShortcutResolver
    metodos:
      - nombre: Resolve
        input: "fieldName string, effective *rules.StemFile"
        output: string
dependencias_externas: []
tests:
  - "estado" resuelve a "frontmatter.estado"
  - "tipo" resuelve a "frontmatter.tipo"
  - "path" queda como "path"
  - "body" resuelve a document body
  - Campo desconocido se trata como literal dot-path
  - Shortcuts de .stem scope override defaults
```

## Dependencias

- T001 completado (query engine usa field paths)
- F02/S001 (effective .stem puede definir shortcuts adicionales)

## Alcance

**In**:
1. Default shortcuts: estado→frontmatter.estado, tipo→frontmatter.tipo, path→path, body→body
2. Funcion `Resolve(fieldName string, effective *StemFile) string`
3. Campos desconocidos se tratan como literal (no error)
4. Integrar con query engine para resolver antes de evaluar

**Out**: Custom shortcut syntax in .stem, complex path expressions

## Estado inicial esperado

- Query engine funcional (T001)
- StemFile disponible

## Criterios de Aceptacion

- `Resolve("estado", nil)` retorna "frontmatter.estado"
- `Resolve("tipo", nil)` retorna "frontmatter.tipo"
- `Resolve("path", nil)` retorna "path"
- `Resolve("frontmatter.custom", nil)` retorna "frontmatter.custom" (passthrough)
- Query engine usa Resolve internamente antes de evaluar condiciones

## Fuente de verdad

- `src/rootline/docs/research/I1-query-operators.md` seccion 9 (Nested field access, shortcuts table)
