---
estado: Specified
tipo: historia
---
# S001: Engine rechaza stems v1 en tiempo de carga

**Feature**: [F01 Hard-reject v1 stems](../README.md)
**Capacidad**: Stems con version 0 o 1 producen error en parse time con mensaje explicativo
**Cubre**: Milestone de F01 — rootline rechaza v1 y todos los tests usan v2

## Antes / Despues

**Antes**: Stems v1 (version: 0 o 1) se parsean con warning. El engine mantiene branches condicionales v1/v2 en rules.go, hierarchy.go, stemhealth.go. 179 test stems usan version: 1.

**Despues**: Stems v1 producen error hard con mensaje: "stem version N is no longer supported". No hay branches condicionales v1/v2. Todos los tests usan version: 2.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline validate` con stem version 0 produce error (no warning)
- [ ] `rootline validate` con stem version 1 produce error (no warning)
- [ ] `go test ./... -race` pasa verde sin ningun stem v1 en tests
- [ ] No existe funcion `rejectLevelsInV2` ni check "version-deprecated"

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV2: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-migrate-test-stems-to-v2.md) | Migrar ~179 test stems de version:1 a version:2 |
| [T002](T002-implement-v1-rejection.md) | Implementar rechazo de version 0/1 en ParseStem |
| [T003](T003-add-rejection-tests.md) | Agregar test de rechazo v1, eliminar tests backward compat |

## Fuente de verdad

- `internal/rules/rules.go` — ParseStem, rejectLevelsInV2
- `internal/rules/hierarchy.go` — branch v1/v2 en linea 15
- `internal/rules/stemhealth.go` — check version-deprecated
- `internal/rules/testdata/*.stem` — 5 test stem files
