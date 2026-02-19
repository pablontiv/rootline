---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Aplicar scope.match del .stem efectivo

**Story**: [S003 File Scanner](README.md)

## Contexto

Cada directorio puede tener un .stem con `scope.match` (ej: `"*.md"`). El scanner debe aplicar este filtro antes de pasar archivos al extractor. Un directorio sin scope hereda el scope del padre via merge.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/index
interfaces:
  - nombre: ScopeFilter
    metodos:
      - nombre: MatchesScope
        input: "filePath string, effectiveStem *rules.StemFile"
        output: bool
dependencias_externas:
  - path/filepath (stdlib — filepath.Match para globs)
tests:
  - "*.md" matchea archivos .md
  - "*.md" no matchea archivos .txt
  - Sin scope definido matchea todo
  - Scope con patron complejo (ej: "T[0-9]*.md")
```

## Dependencias

- T001 (scanner funcional) + F02/S001 (effective .stem disponible)

## Alcance

**In**:
1. Funcion `MatchesScope(filePath string, effectiveStem *rules.StemFile) bool`
2. Usa `filepath.Match` para evaluar `scope.match` pattern
3. Sin scope definido = match everything
4. Integrar con Scanner.Scan para filtrar antes de extraccion

**Out**: Custom scope patterns, multiple match patterns

## Estado inicial esperado

- Scanner funcional (T001)
- StemFile con Scope disponible (F02/S001)

## Criterios de Aceptacion

- `MatchesScope("doc.md", stem{scope.match:"*.md"})` retorna true
- `MatchesScope("data.json", stem{scope.match:"*.md"})` retorna false
- `MatchesScope("doc.md", stem{scope:nil})` retorna true (no scope = match all)
- Scanner integrado filtra archivos por scope antes de extraction

## Fuente de verdad

- `src/rootline/docs/research/I7-extractors-architecture.md` seccion 6 (Pipeline Integration, scope matching)
- `src/rootline/docs/research/I5-describe-contract.md` (scope in effective schema)
