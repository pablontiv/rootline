---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Seleccion determinista de stem por riqueza de schema

**Story**: [S001 Fix Priority Conflicts](README.md)

## Contexto

`runFixAll` en fix.go itera `effectiveStems` (un `map[string]*rules.StemFile`) y toma el primer valor con `break`. Go maps no garantizan orden de iteracion, asi que desde repo root puede tomar un stem sin schema enum (ej: el `.stem` de un subdirectorio con solo `id`), perdiendo las definiciones de `estado`, `tipo`, etc.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
archivo: cmd/rootline/fix.go
funciones_afectadas:
  - runFixAll (lineas 191-196, dry-run path)
  - runFixAll (lineas 206-211, apply path)
cambio: |
  Reemplazar patron "first from map":
    var effective *rules.StemFile
    for _, s := range effectiveStems { effective = s; break }
  Con seleccion por riqueza:
    var effective *rules.StemFile
    for _, s := range effectiveStems {
      if effective == nil || len(s.Schema) > len(effective.Schema) {
        effective = s
      }
    }
tests:
  - fix --all --dry-run desde repo root genera extend_enum proposals
  - fix --all --dry-run desde docs/epics/ genera mismos proposals que desde root
```

## Alcance

**In**: Cambiar seleccion de stem en 2 lugares de runFixAll
**Out**: Cambios en MergeStemFiles o WalkUp

## Estado inicial esperado

- `effectiveStems` se puebla correctamente con stems de todos los records
- Desde repo root, el stem con schema mas rico tiene `estado` y `tipo` definidos

## Criterios de Aceptacion

- `rootline fix --all --dry-run -o json` desde repo root incluye proposals tipo `extend_enum`
- Mismos tipos de proposals desde repo root y desde `docs/epics/`
- `go test ./cmd/rootline/...` pasa

## Fuente de verdad

- `cmd/rootline/fix.go` lineas 189-212
