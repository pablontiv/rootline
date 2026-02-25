---
estado: Pending
tipo: feature
---
# F02: Caller Migration and Tooling

**Epic**: [E09](../README.md)
**Satisface**: P1, P3
**Objetivo**: Todos los callers que validan/fijan records individuales usan ResolveForRecord, y las herramientas de diagnostico soportan levels
**Beneficio**: Levels funciona end-to-end en validate, fix, MCP tools, stemhealth, e infer
**Milestone**: `go test ./... -race` pasa y `rootline validate --all docs/epics/` usa levels correctamente

## Scope

**In**: Migrar validate.go, fix.go, mcp/tools.go a ResolveForRecord; stemhealth checks para levels; infer --hierarchy genera levels format; E2E test
**Out**: Core engine (F01), migration de .stem files reales (F03)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-validate-fix-mcp-caller-migration/) | Validate/Fix/MCP Caller Migration | Los pipelines de validate, fix y MCP tools resuelven levels correctamente para cada record |
| [S002](S002-describe-infer-and-stemhealth/) | Describe, Infer and Stemhealth | stemhealth detecta levels malformados e infer genera formato levels |

## Invariantes

- INV1 (heredado): Todos los tests existentes pasan sin modificacion
- INV2 (heredado): Coverage se mantiene >= 85%
- INV4: Callers sin record path (describe, tree) siguen funcionando sin cambios

## Dependencias

- F01: ResolveForRecord y ExpandLevels deben existir antes de migrar callers

## Fuente de verdad

- `cmd/rootline/validate.go` — runValidateAll, runValidate
- `cmd/rootline/fix.go` — fix pipeline
- `internal/mcp/tools.go` — MCP validate/fix tools
- `internal/rules/stemhealth.go` — stem diagnostics
- `internal/infer/hierarchy.go` — hierarchy inference
