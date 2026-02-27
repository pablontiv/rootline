# S001: Code cleanup

**Feature**: [F04 Levels removal](../README.md)
**Capacidad**: All levels-related code removed from the codebase
**Cubre**: The code removal side of the F04 milestone

## Antes / Despues

**Antes**: `HierarchyLevel` struct, `ExpandLevels`, `CheckNesting`, `matchLevel`, `containsString`, `mergeLevels`, and stemhealth checks 8-9 (`levels-children-valid`, `levels-no-cycles`) exist in the codebase. `ResolveForRecord` has dual-path logic for v1 and v2. CLI commands call `CheckNesting`.

**Despues**: All removed. `ResolveForRecord` only supports v2 match resolution. `StemFile` has no `Levels` field. No `levels:` parsing in `ParseStem`. CLI commands don't call `CheckNesting`. Stemhealth has no levels-specific checks.

## Criterios de Aceptacion (semanticos)

- [ ] `grep -r "HierarchyLevel\|ExpandLevels\|CheckNesting\|mergeLevels" internal/ cmd/` returns zero matches
- [ ] `go build ./cmd/rootline/` succeeds
- [ ] All tests pass

## Invariantes

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-remove-hierarchylevel-and-expandlevels.md) | Remove HierarchyLevel struct and ExpandLevels |
| [T002](T002-remove-stemhealth-checks-and-mergelevels.md) | Remove stemhealth checks 8-9 and mergeLevels |
| [T003](T003-remove-checknesting-from-cli.md) | Remove CheckNesting from CLI call sites |

## Fuente de verdad

- `internal/rules/rules.go` — HierarchyLevel, StemFile.Levels
- `internal/rules/hierarchy.go` — ExpandLevels, CheckNesting
- `internal/rules/merge.go` — mergeLevels
- `internal/rules/stemhealth.go` — Checks 8-9
- `cmd/rootline/validate.go` — CheckNesting calls
- `cmd/rootline/fix.go` — CheckNesting call
