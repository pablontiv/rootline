---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Migrate validate.go callers to ResolveForRecord

**Story**: [S001 Validate/Fix/MCP Caller Migration](README.md)
**Contribuye a**: validate.go (single file + batch) usa ResolveForRecord

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV4: Callers sin record path (describe, tree) no cambian
  - Verificar: `git diff cmd/rootline/describe.go cmd/rootline/tree.go` shows no changes

## Contexto

`cmd/rootline/validate.go` has multiple call sites that use `WalkUp + MergeStemFiles` to get the effective schema for validating records. These need to be replaced with `ResolveForRecord` to support levels expansion.

The key call sites are:
- Single file validation (~line 94): validates one file
- Batch validation in `runValidateAll` (~line 142): iterates over files in DeriveAllSimple
- Per-file validation in the batch loop (~line 169): validates each file
- Post-derive validation (~line 185)

`ResolveForRecord(dir, recordPath)` is a drop-in replacement that adds levels expansion when the stem has `levels:`.

## Dependencias

- F01/S001/T003: ResolveForRecord function must exist

## Alcance

**In**:
1. Replace `WalkUp + MergeStemFiles` calls in `cmd/rootline/validate.go` with `ResolveForRecord`
2. Ensure the record path is correctly computed relative to the stem root
3. Update imports if needed
4. Verify all existing validate tests pass

**Out**: fix.go migration (T002), MCP tools (T002), describe/tree (explicitly not migrated)

## Estado inicial esperado

- `ResolveForRecord` exists in `internal/rules/hierarchy.go` (from F01)
- `cmd/rootline/validate.go` uses WalkUp+MergeStemFiles at lines ~94, ~142, ~169, ~185

## Criterios de Aceptacion

- `go test ./cmd/rootline/ -run TestValidate -v` passes
- `rootline validate <single-file>` works correctly with and without levels
- `rootline validate --all <dir>` works correctly with and without levels
- No calls to `WalkUp + MergeStemFiles` remain in validate.go for per-record validation
- `go test ./... -race` passes

## Fuente de verdad

- `cmd/rootline/validate.go` — validate pipeline
- `internal/rules/hierarchy.go` — ResolveForRecord
