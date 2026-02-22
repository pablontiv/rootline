---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Cambiar target patterns de filepath.Match glob a regexp

**Story**: [S004 Graph Schema-Aware Link Filtering](README.md)

[[blocks:T001-graph-load-stem-filter-links]]

## Contexto

`validateLinks()` en `internal/rules/validate.go:248` usa `filepath.Match(rule.Target, link.Target)` para validar targets contra el pattern definido en .stem. `filepath.Match` es un glob limitado — soporta `*`, `?`, `[lo-hi]` pero no `\d`, `+`, alternacion ni anchors. Esto obliga a patterns imprecisos como `T*` (matchea cualquier cosa que empiece con T).

Cambiar a `regexp.MatchString` (Go stdlib) permite patterns precisos como `^T\d{3}-` que matchea exactamente task IDs (T001-nombre, T002-otro) y rechaza texto casual (target, table, test).

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: validateLinks (function existente)
    metodos:
      - nombre: validateLinks
        input: "links []extract.Link, schema LinkSchema, source string"
        output: "[]ValidationError"
dependencias_externas: []
tests:
  - Link target "T001-task-name" con pattern "^T\\d{3}-" no genera error
  - Link target "target" con pattern "^T\\d{3}-" genera error
  - Link target "A,B,C,A" con pattern "^T\\d{3}-" genera error
  - Regexp invalido en .stem genera error descriptivo (no panic)
  - Sin target rule → no se valida target (permisivo, igual que antes)
```

## Dependencias

- T001 (graph carga .stem) — el graph necesita usar .stem para que el pattern tenga efecto en graph --check
- `internal/rules/validate.go` — validateLinks() existente (modificar)

## Alcance

**In**:
1. En `internal/rules/validate.go:248`: cambiar `filepath.Match(rule.Target, link.Target)` → `regexp.MatchString(rule.Target, link.Target)`
2. Agregar import `regexp` (ya no se necesita `path/filepath` para esto)
3. Manejar error de regexp invalido: si `regexp.MatchString` retorna error (pattern invalido), generar ValidationError descriptivo en vez de panic
4. En `docs/epics/.stem:48`: cambiar `target: "T*"` a `target: "^T\\d{3}-"`
5. Actualizar tests en `internal/rules/validate_test.go` para usar regex patterns en vez de globs
6. Verificar que `rootline validate docs/epics/` sigue pasando con archivos existentes

**Out**: Cambios a LinkRule struct, migracion automatica de globs existentes a regex, expr-lang/expr integration (reservado para futuras validaciones cross-field)

## Estado inicial esperado

- `internal/rules/validate.go:248` usa `filepath.Match`
- `docs/epics/.stem:48` tiene `target: "T*"`
- Tests en `validate_test.go` usan glob patterns en assertions
- T001 ya implementado (graph carga .stem)

## Criterios de Aceptacion

- `regexp.MatchString("^T\\d{3}-", "T001-task-name")` → true (match)
- `regexp.MatchString("^T\\d{3}-", "target")` → false (no match)
- `regexp.MatchString("^T\\d{3}-", "A,B,C,A")` → false (no match)
- `regexp.MatchString("^T\\d{3}-", "T99-short")` → false (solo 2 digitos)
- .stem con regexp invalido `target: "[invalid"` → ValidationError, no panic
- `go test ./internal/rules/ -run TestValidateLinks -v` pasa con nuevos patterns
- `rootline validate docs/epics/` → 0 errores (archivos existentes validos con nuevo pattern)
- `go build ./cmd/rootline/ && rootline graph --check docs/epics/` → 0 broken links
- `go vet ./...` limpio

## Fuente de verdad

- `internal/rules/validate.go:228-261` — validateLinks() (cambiar filepath.Match → regexp)
- `internal/rules/validate_test.go:359-430` — tests de link validation (actualizar patterns)
- `docs/epics/.stem:45-48` — links section (actualizar target pattern)
