---
estado: Completed
tipo: refactor
ejecutable_en: 1 sesion
---
# T001: Eliminar archivos de migracion v1 y sus tests

**Story**: [S001 Codigo de migracion v1 eliminado](README.md)
**Contribuye a**: `golangci-lint run ./...` no reporta dead code de migracion v1

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV2: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Con el rechazo de v1 implementado (F01), las funciones `ConvertLevelsToMatch()` y `UpgradeToV2()` son dead code. Sus archivos y tests deben eliminarse. Puede haber imports o referencias en otros archivos del paquete `internal/migrate/` que necesiten limpieza.

## Alcance

**In**:
1. Eliminar `internal/migrate/levels_to_match.go`
2. Eliminar `internal/migrate/to_v2.go`
3. Eliminar `internal/migrate/levels_to_match_test.go`
4. Eliminar `internal/migrate/to_v2_test.go`
5. Limpiar imports y referencias rotas en otros archivos de `internal/migrate/`

**Out**: No tocar CLI flags (eso es T002). No tocar docs (eso es T003).

## Estado inicial esperado

- F01 completado: v1 rechazado en ParseStem
- Los archivos a eliminar existen y compilan pero su codigo es inalcanzable

## Criterios de Aceptacion

- `ls internal/migrate/levels_to_match.go internal/migrate/to_v2.go 2>&1` retorna "No such file"
- `ls internal/migrate/levels_to_match_test.go internal/migrate/to_v2_test.go 2>&1` retorna "No such file"
- `go build ./...` compila sin errores
- `golangci-lint run ./...` sin dead code warnings

## Fuente de verdad

- `internal/migrate/levels_to_match.go` — v1Level, v1StemFile, ConvertLevelsToMatch
- `internal/migrate/to_v2.go` — UpgradeToV2, ToV2Result
- `internal/migrate/levels_to_match_test.go`, `internal/migrate/to_v2_test.go`
