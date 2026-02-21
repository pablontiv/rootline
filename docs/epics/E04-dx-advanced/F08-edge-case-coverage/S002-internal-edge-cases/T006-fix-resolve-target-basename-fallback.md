---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T006: Fix resolveTarget — agregar fallback por basename para wiki-links cross-directory

**Story**: [S002 Internal Edge Cases](README.md)

## Contexto

`internal/graph/graph.go:resolveTarget` resuelve targets de wiki-links relativos al directorio del source. Si el target no contiene `/` ni `..`, lo devuelve tal cual. Esto funciona para links dentro del mismo directorio (ej: `[[blocks:T002-second.md]]` desde el mismo dir), pero falla para links cross-directory.

Ejemplo concreto: en `F09/S001/T002-implement-validate-directory.md` el wiki-link `[[blocks:T001-extend-stemfile-structural-types]]` genera target `T001-extend-stemfile-structural-types`. Pero el nodo real en el grafo es `S001-structural-directory-rules/T001-extend-stemfile-structural-types.md`. No hay match → broken link falso positivo.

## Alcance

**In**:
1. En `internal/graph/graph.go`, modificar `Build()` para que despues de crear edges con `resolveTarget`, haga un segundo paso de resolucion: si el target no matchea ningun nodo, buscar por basename (con y sin `.md`) entre todos los nodos del grafo
2. Si hay exactamente 1 match por basename → reescribir el target del edge al path completo del nodo
3. Si hay 0 o >1 matches → dejar el target como esta (broken link legitimo o ambiguo)
4. Agregar tests en `internal/graph/graph_test.go`:
   - `TestResolveTarget_BasenameFallback` — target sin path matchea nodo con subdirectorio
   - `TestResolveTarget_BasenameFallback_WithExtension` — target `T001-name.md` matchea `subdir/T001-name.md`
   - `TestResolveTarget_BasenameFallback_Ambiguous` — target matchea 2 nodos → sigue como broken link
   - `TestResolveTarget_BasenameFallback_NoMatch` — target no matchea nada → broken link

**Out**: Cambios a `resolveTarget` signature. Cambios a extract/links.go. Cambios al .stem.

## Estado inicial esperado

- `internal/graph/graph.go` tiene `resolveTarget` que solo resuelve paths con `/` o `..`
- `rootline graph --check` en F09 reporta 5 broken links falsos positivos por targets cross-directory
- `go test ./internal/graph/ -race` pasa

## Criterios de Aceptacion

- `go test ./internal/graph/ -race` pasa con los 4 tests nuevos
- `rootline graph docs/epics/E04-dx-advanced/F09-planning-structure-validation/ --check` no reporta broken links para los wiki-links cross-directory existentes
- `go vet ./...` sin errores
- Coverage de `internal/graph/` >= 85%

## Fuente de verdad

- `internal/graph/graph.go` — Build() y resolveTarget()
- `internal/graph/graph_test.go` — tests existentes
