---
estado: Specified
tipo: test
ejecutable_en: 1 sesion
---
# T001: Migrar test stems de v1 a v2

**Story**: [S001 Engine rechaza stems v1](README.md)
**Contribuye a**: `go test ./... -race` pasa verde sin ningun stem v1 en tests

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`

## Contexto

Rootline tiene ~179 inline test stems con `version: 1` y 5 archivos `.stem` en `internal/rules/testdata/` con `version: 1`. Todos deben migrarse a `version: 2` antes de implementar el rechazo de v1 en ParseStem (T002). Los stems v2 ya parsean identicamente que v1 con el engine actual, asi que cambiar la version no rompe nada.

## Alcance

**In**:
1. Cambiar `version: 1` a `version: 2` en todos los test stems inline en archivos `*_test.go`
2. Cambiar `version: 1` a `version: 2` en los 5 archivos de `internal/rules/testdata/*.stem`
3. Cambiar stems sin version (version: 0 implicito) a `version: 2` en tests

**Out**: No modificar logica de parsing ni agregar/eliminar tests (eso es T002 y T003)

## Estado inicial esperado

- `go test ./... -race` pasa verde con stems v1
- Stems v2 parsean identicamente que v1 (backward compat vigente)

## Criterios de Aceptacion

- `grep -rn "version: 1" internal/rules/testdata/` retorna 0 resultados
- `grep -rn '"version: 1"' cmd/rootline/*_test.go internal/rules/*_test.go internal/migrate/*_test.go internal/infer/*_test.go` retorna 0 resultados (excepto en tests que prueban especificamente el rechazo de v1, que se crean en T003)
- `go test ./... -race` pasa verde

## Fuente de verdad

- `internal/rules/testdata/docs.stem`, `epics.stem`, `prd.stem`, `research.stem`, `task.stem`
- `cmd/rootline/describe_test.go`, `fix_test.go`, `migrate_test.go`
- `internal/rules/rules_test.go`, `hierarchy_test.go`, `stemhealth_test.go`
- `internal/migrate/source_test.go`
- `internal/infer/hierarchy_test.go`
