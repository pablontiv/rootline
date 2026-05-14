---
estado: Specified
tipo: task
---
# T005: Remove `title` magic from `query.go`

**Outcome**: [O14 Field-agnostic refactor](README.md)
**Contribuye a**: query.go no tiene conocimiento especial de ningún campo; titulo llega via campo derivado como cualquier otro

## Preserva

- INV1: `rootline query --select titulo` retorna el mismo valor que antes (el H1 del archivo)
  - Verificar: `rootline query /home/shared/rootline/docs/roadmap --select path,titulo --output json`
- INV2: Otros campos en --select siguen funcionando igual
  - Verificar: `rootline query /home/shared/rootline/docs/roadmap --select path,estado,tipo --output json`

## Contexto

`cmd/rootline/query.go` tiene un `case "title":` en el switch de `projectQueryResult` que llama `extractTitle()`, una función que extrae el H1 del body directamente. Con T001 implementado, `titulo` llega en `rec.Derived` vía `source: body.h1`, por lo que el case especial y la función son redundantes y deben eliminarse.

## Alcance

**In**:
1. Eliminar `case "title":` del switch en `projectQueryResult`
2. Eliminar la función `extractTitle()` de query.go

**Out**:
- No cambiar el comportamiento de ningún otro campo en --select
- No modificar la lógica de filtrado (`--where`)

## Estado inicial esperado

- T001 completada (titulo se deriva vía source: body.h1 y llega en rec.Derived)
- `case "title":` y `extractTitle()` existen en `cmd/rootline/query.go`

## Criterios de Aceptación

- `case "title":` eliminado del switch en `projectQueryResult`
- `extractTitle()` eliminada de query.go
- `rootline query /home/shared/rootline/docs/roadmap --select path,titulo --output json` sigue retornando campo `titulo` con el H1
- `go test ./cmd/rootline/...` verde

## Fuente de verdad

- `cmd/rootline/query.go` — projectQueryResult, extractTitle
