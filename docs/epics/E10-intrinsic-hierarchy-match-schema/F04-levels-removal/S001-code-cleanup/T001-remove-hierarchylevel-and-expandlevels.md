---
estado: Specified
tipo: refactor
ejecutable_en: 1 sesion
---
# T001: Remove HierarchyLevel struct and ExpandLevels

**Story**: [S001 Code cleanup](README.md)
**Contribuye a**: All levels-related code removed

## Preserva

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

With F01-F03 complete, all stems use v2 format and the v1 code path is no longer needed. This task removes the core `levels:` data structures and functions from `internal/rules/`.

## Especificacion Tecnica

In `internal/rules/rules.go`:
- Delete `HierarchyLevel` struct (lines 17-22)
- Remove `Levels` field from `StemFile` (line 35)
- Remove v1 parsing branch in `ParseStem` (if any v1-specific code remains)

In `internal/rules/hierarchy.go`:
- Delete `ExpandLevels` function
- Delete `matchLevel` helper
- Delete `containsString` helper
- Remove v1 branch from `ResolveForRecord`

Update/remove tests in `internal/rules/hierarchy_test.go` that test v1-specific functions.

## Dependencias

- F01, F02, F03 all completed
- F03/S002/T002: project stems migrated to v2

## Alcance

**In**:
1. Delete HierarchyLevel struct
2. Remove StemFile.Levels field
3. Delete ExpandLevels, matchLevel, containsString
4. Remove v1 branch from ResolveForRecord
5. Update/delete affected tests

**Out**: Stemhealth checks (T002), CLI call sites (T003)

## Estado inicial esperado

- All stems use v2 format
- v1 code path is dead code

## Criterios de Aceptacion

- `grep -r "HierarchyLevel\|ExpandLevels\|matchLevel\|containsString" internal/rules/` returns zero matches
- `go build ./cmd/rootline/` succeeds
- `go test ./internal/rules/ -race` passes
- `go vet ./...` passes

## Fuente de verdad

- `internal/rules/rules.go` — HierarchyLevel (lines 17-22), StemFile.Levels (line 35)
- `internal/rules/hierarchy.go` — ExpandLevels (lines 56-86), matchLevel, containsString
