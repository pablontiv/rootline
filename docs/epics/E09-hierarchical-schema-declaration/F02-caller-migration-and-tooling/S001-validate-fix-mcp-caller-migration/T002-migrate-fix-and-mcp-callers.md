---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Migrate fix.go and MCP tools callers

**Story**: [S001 Validate/Fix/MCP Caller Migration](README.md)
**Contribuye a**: fix.go y MCP tools usan ResolveForRecord

[[blocks:T001-migrate-validate-callers]]

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV4: Callers sin record path no cambian
  - Verificar: describe.go and tree.go unchanged

## Contexto

After migrating validate.go (T001), the same pattern needs to be applied to:

1. `cmd/rootline/fix.go` — the fix pipeline that generates and applies proposals for validation errors. It uses WalkUp+MergeStemFiles to get the effective schema, then generates fix proposals.

2. `internal/mcp/tools.go` — the MCP server tools for validate and fix. These call the same core pipeline but via JSON-RPC. They also use WalkUp+MergeStemFiles.

Both should switch to `ResolveForRecord` for per-record schema resolution.

## Dependencias

- T001: validate.go migration pattern established
- F01/S001/T003: ResolveForRecord exists

## Alcance

**In**:
1. Replace WalkUp+MergeStemFiles in `cmd/rootline/fix.go` with ResolveForRecord
2. Replace WalkUp+MergeStemFiles in `internal/mcp/tools.go` (validate + fix tool handlers) with ResolveForRecord
3. Verify existing tests pass for both

**Out**: E2E test (T003), stemhealth/infer (S002)

## Estado inicial esperado

- ResolveForRecord exists (F01)
- validate.go already migrated (T001)
- fix.go and mcp/tools.go use WalkUp+MergeStemFiles

## Criterios de Aceptacion

- `go test ./cmd/rootline/ -run TestFix -v` passes
- `go test ./internal/mcp/ -v` passes
- `rootline fix <file>` works correctly with and without levels
- MCP validate/fix tools work via JSON-RPC with levels
- `go test ./... -race` passes

## Fuente de verdad

- `cmd/rootline/fix.go` — fix pipeline
- `internal/mcp/tools.go` — MCP tool handlers
- `internal/rules/hierarchy.go` — ResolveForRecord
