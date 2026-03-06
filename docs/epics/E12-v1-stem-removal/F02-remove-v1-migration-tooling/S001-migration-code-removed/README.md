---
estado: Completed
tipo: historia
---
# S001: Codigo de migracion v1 eliminado del binario

**Feature**: [F02 Remove v1 migration tooling](../README.md)
**Capacidad**: No existe codigo de migracion v1→v2 en el binario ni en la documentacion
**Cubre**: Milestone de F02 — flags CLI eliminados, archivos borrados, docs actualizados

## Antes / Despues

**Antes**: El binario incluye `levels_to_match.go` (ConvertLevelsToMatch), `to_v2.go` (UpgradeToV2), y el CLI tiene flags `--from-levels` y `--to-v2`. Documentacion en levels.md y migrate.md referencia migracion v1.

**Despues**: Archivos eliminados, flags eliminados, documentacion actualizada. `rootline migrate --help` no muestra opciones v1. `golangci-lint` no reporta dead code.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline migrate --help` no muestra `--from-levels` ni `--to-v2`
- [ ] `golangci-lint run ./...` no reporta dead code de migracion v1
- [ ] `grep -r "from-levels\|to-v2\|v1.*supported" docs/` no encuentra referencias a v1 como soportado

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV2: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-delete-migration-files.md) | Eliminar archivos de migracion v1 y sus tests |
| [T002](T002-remove-cli-flags.md) | Eliminar flags --from-levels/--to-v2 del CLI |
| [T003](T003-update-documentation.md) | Actualizar documentacion (levels.md, migrate.md, research) |

## Fuente de verdad

- `internal/migrate/levels_to_match.go`, `internal/migrate/to_v2.go`
- `cmd/rootline/migrate.go`
- `docs/levels.md`, `docs/migrate.md`
