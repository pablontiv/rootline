---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T005: Agregar ScopeResolver a tree command

**Story**: [S001 Fix Priority Conflicts](README.md)

## Contexto

`runTree` en tree.go llama `index.Scan(absRoot, reg)` sin `WithScopeResolver`. Esto significa que `tree` no respeta el `scope.match` del `.stem`, incluyendo todos los archivos .md del directorio (incluyendo archivos fuera del scope del roadmap como CHANGELOG.md, CONTRIBUTING.md, etc).

`validate.go` y `fix.go` ya usan `WithScopeResolver` — tree es el unico comando que no.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
archivo: cmd/rootline/tree.go
funcion: runTree
cambio: |
  1. Agregar import de "github.com/pablontiv/rootline/internal/rules"
  2. Crear resolver closure (mismo patron que fix.go y validate.go):
     resolver := func(dir string) *rules.StemFile {
       entries, err := rules.WalkUp(dir)
       if err != nil { return nil }
       return rules.MergeStemFiles(entries)
     }
  3. Pasar resolver a Scan:
     records, err := index.Scan(absRoot, reg, index.WithScopeResolver(resolver))
tests:
  - tree desde repo root no incluye archivos fuera de scope
  - tree desde docs/epics/ muestra solo archivos del roadmap
```

## Alcance

**In**: Agregar ScopeResolver a tree.go
**Out**: Cambios en index.Scan o ScopeResolver API

## Estado inicial esperado

- `index.WithScopeResolver` existe y funciona (usado en fix.go y validate.go)
- `.stem` tiene `scope.match` configurado

## Criterios de Aceptacion

- `rootline tree . -o table` desde repo root no incluye CHANGELOG.md, CONTRIBUTING.md, etc
- `rootline tree docs/epics -o table` muestra solo archivos del scope del roadmap
- `go test ./cmd/rootline/...` pasa

## Fuente de verdad

- `cmd/rootline/tree.go` funcion runTree
- `cmd/rootline/validate.go` (patron de referencia)
- `cmd/rootline/fix.go` (patron de referencia)
