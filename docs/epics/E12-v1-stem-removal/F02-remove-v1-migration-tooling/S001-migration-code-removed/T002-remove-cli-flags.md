---
estado: Completed
tipo: refactor
ejecutable_en: 1 sesion
---
# T002: Eliminar flags --from-levels/--to-v2 del CLI

**Story**: [S001 Codigo de migracion v1 eliminado](README.md)
**Contribuye a**: `rootline migrate --help` no muestra --from-levels ni --to-v2

[[blocks:T001-delete-migration-files]]

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV2: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Con los archivos de migracion eliminados (T001), los flags CLI que los invocan deben eliminarse tambien. El archivo `cmd/rootline/migrate.go` tiene flags `--from-levels` y `--to-v2` con funciones handler `runMigrateFromLevels()` y `runMigrateToV2()`. Tests asociados en `cmd/rootline/migrate_test.go` tambien deben eliminarse.

## Alcance

**In**:
1. Eliminar flag `--from-levels` y funcion `runMigrateFromLevels()` de `cmd/rootline/migrate.go`
2. Eliminar flag `--to-v2` y funcion `runMigrateToV2()` de `cmd/rootline/migrate.go`
3. Eliminar tests asociados a estos flags en `cmd/rootline/migrate_test.go`
4. Limpiar imports no usados

**Out**: No tocar otros flags de migrate (--split, etc.). No tocar docs (eso es T003).

## Estado inicial esperado

- T001 completado: archivos de migracion eliminados
- `go build ./...` falla por imports rotos en migrate.go (referencia a funciones eliminadas)

## Criterios de Aceptacion

- `rootline migrate --help` no muestra `--from-levels` ni `--to-v2`
- `grep -n "from-levels\|to-v2\|runMigrateFromLevels\|runMigrateToV2" cmd/rootline/migrate.go` retorna 0 resultados
- `go build ./cmd/rootline/` compila
- `go test ./cmd/rootline/ -race -run TestMigrate` pasa verde

## Fuente de verdad

- `cmd/rootline/migrate.go` — flags y handlers
- `cmd/rootline/migrate_test.go` — tests de --from-levels y --to-v2
