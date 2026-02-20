---
estado: Pending
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: merge_test.go — null removal, herencia 4 niveles, override required

**Story**: [S002 Internal Edge Cases](README.md)

## Contexto

La semantica de merge de rootline dice que `null` en un child elimina el campo del parent. Los tests existentes cubren merge basico pero no verifican la semantica de null removal. Tampoco se testea herencia de 4+ niveles (que es el caso real en docs/epics/Epic/Feature/Story/Task).

## Alcance

**In**: Agregar a `internal/rules/merge_test.go`:
1. `TestMerge_NullRemovesField` — stem A define `campo: {required: true}`, stem B define `campo: null` → el merge result NO incluye `campo`
2. `TestMerge_FourLevelChain` — 4 stems en cadena (root→epic→feature→story), cada uno agrega un campo; el result tiene todos los campos de los 4 niveles
3. `TestMerge_ChildOverridesRequired` — parent: `required: true`, child: `required: false` → result: `required: false`
4. `TestMerge_NullValidateRule` — parent tiene una validate rule, child tiene `null` para esa rule → rule eliminada del result

**Out**: Cambios a merge.go, tests de otros packages

## Estado inicial esperado

- `internal/rules/merge_test.go` existe con tests de merge basico
- La semantica de null removal esta implementada en merge.go
- `go test ./internal/rules/ -race` pasa

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestMerge_NullRemovesField -v` pasa
- `go test ./internal/rules/ -run TestMerge_FourLevelChain -v` pasa
- `go test ./internal/rules/ -run TestMerge_ChildOverridesRequired -v` pasa
- `go test ./internal/rules/ -race` pasa sin regresiones

## Fuente de verdad

- `internal/rules/merge_test.go` — archivo a extender
- `internal/rules/merge.go` — implementacion del merge con null semantics
