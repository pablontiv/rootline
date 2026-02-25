---
estado: Pending
---
# S001: Validate/Fix/MCP Caller Migration

**Feature**: [F02 Caller Migration and Tooling](../README.md)
**Capacidad**: Los pipelines de validate, fix y MCP tools resuelven levels correctamente para cada record individual
**Cubre**: Milestone de F02 — validate/fix/MCP usan ResolveForRecord

## Antes / Despues

**Antes**: `validate.go`, `fix.go` y `mcp/tools.go` llaman `WalkUp + MergeStemFiles` directamente. Esto funciona con child `.stem` files reales pero no con `levels:` (no expande levels a virtual entries).

**Despues**: Los tres callers usan `ResolveForRecord(dir, recordPath)` que internamente hace WalkUp + Merge + ExpandLevels si hay levels. El effective schema es correcto tanto con child `.stem` como con `levels:`.

## Criterios de Aceptacion (semanticos)

- [ ] `validate.go` (single file + batch) usa ResolveForRecord
- [ ] `fix.go` usa ResolveForRecord en su loop
- [ ] `mcp/tools.go` validate y fix tools usan ResolveForRecord
- [ ] E2E test con fixture de levels verifica pipeline completo
- [ ] Callers sin record path (describe, tree) no cambian

## Invariantes

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV4: Callers sin record path no cambian
  - Verificar: `describe.go` y `tree.go` sin diff

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-migrate-validate-callers.md) | Migrate validate.go callers to ResolveForRecord |
| [T002](T002-migrate-fix-and-mcp-callers.md) | Migrate fix.go and MCP tools callers |
| [T003](T003-e2e-integration-test.md) | E2E integration test with levels fixture |

## Fuente de verdad

- `cmd/rootline/validate.go` — lines 94, 142, 169, 185 (WalkUp+Merge callers)
- `cmd/rootline/fix.go` — fix loop
- `internal/mcp/tools.go` — validate/fix MCP tools
