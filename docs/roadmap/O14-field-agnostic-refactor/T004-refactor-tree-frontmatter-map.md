---
estado: Specified
tipo: task
---
# T004: Refactor `tree.go` — Frontmatter map, remove Completed/Estado hardcodings

**Outcome**: [O14 Field-agnostic refactor](README.md)
**Contribuye a**: el output de `rootline tree` es agnóstico al schema; roadmapctl puede leer cualquier campo sin que rootline lo conozca

## Preserva

- INV1: `--where` sigue filtrando correctamente sobre cualquier campo
  - Verificar: `rootline tree /home/shared/rootline/docs/roadmap --where "not (estado in ['Completed', 'Obsolete'])" --output json | jq '.root.total'`
- INV2: roadmapctl puede leer `frontmatter.estado` y `frontmatter.tipo` del output de tree
  - Verificar: `rootline tree /home/shared/rootline/docs/roadmap --output json | jq '.root.children[0].children[0].frontmatter'`

## Contexto

`treeNode` en `cmd/rootline/tree.go` tiene campos `Completed int` y `Estado string json:"estado"` que hardcodean conocimiento de dominio. `buildTree` usa `if estado == "Completed"` para contar nodos completados. Hay un fallback `estadoField := "estado"` que busca el campo por domain lookup.

El cambio reemplaza `Completed int` y `Estado string` con `Frontmatter map[string]any json:"frontmatter,omitempty"` y popula ese map con todos los campos del record (frontmatter + derived). roadmapctl heredará `titulo`, `estado`, `tipo` directamente del map. El output JSON versiona como `version: 2`.

Requiere T001 para que `titulo` llegue vía campo derivado en `rec.Derived`.

## Alcance

**In**:
1. Reemplazar `treeNode.Completed int` y `treeNode.Estado string` con `treeNode.Frontmatter map[string]any json:"frontmatter,omitempty"`
2. Eliminar `if estado == "Completed"` y la lógica de conteo `Completed`
3. Eliminar `estadoField := "estado"` fallback y cualquier domain lookup en tree.go
4. `buildTree` popula `leaf.Frontmatter` con frontmatter + derived fields del record
5. Output JSON: campo `version: 2` en la raíz del resultado

**Out**:
- No cambiar el comportamiento de `--where` (ya funciona como filtro)
- No tocar roadmapctl todavía

## Estado inicial esperado

- T001 completada (titulo se deriva via source: body.h1)
- `go test ./...` pasa

## Criterios de Aceptación

- `treeNode` no tiene campos `Completed` ni `Estado`
- `treeNode.Frontmatter map[string]any` existe y se serializa como `"frontmatter": {...}` en JSON
- `rootline tree <roadmap> --output json | jq '.root.children[0].children[0].frontmatter.estado'` retorna el valor de estado de la task
- `rootline tree <roadmap> --where "not (estado in ['Completed', 'Obsolete'])" --output json` filtra correctamente
- `go test ./cmd/rootline/... ./internal/...` verde

## Fuente de verdad

- `cmd/rootline/tree.go` — treeNode, buildTree
