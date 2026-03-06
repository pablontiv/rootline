---
estado: Completed
tipo: feature
---
# F02: Remove V1 Migration Tooling

**Epic**: [E12 V1 Stem Removal](../README.md)
**Satisface**: P2, P4
**Objetivo**: Eliminar del binario todo el codigo de migracion v1→v2 y actualizar documentacion
**Beneficio**: Reduce tamano del binario, elimina dead code, documentacion refleja estado real
**Milestone**: `--from-levels`/`--to-v2` no existen en CLI; archivos de migracion eliminados; docs actualizados

## Scope

**In**: Eliminar levels_to_match.go, to_v2.go, flags CLI, y documentacion v1
**Out**: Cambios al engine core (eso fue F01)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Migration code removed](S001-migration-code-removed/) | No existe codigo ni CLI de migracion v1 |

## Invariantes

- INV1 (heredado): `go test ./... -race` pasa verde en cada commit
- INV2 (heredado): Coverage ≥85%

## Dependencias

- F01 debe completarse primero (el codigo de migracion solo es dead code despues del rechazo)

## Fuente de verdad

- `internal/migrate/levels_to_match.go` — ConvertLevelsToMatch
- `internal/migrate/to_v2.go` — UpgradeToV2
- `cmd/rootline/migrate.go` — flags --from-levels, --to-v2
- `docs/levels.md`, `docs/migrate.md` — documentacion v1
