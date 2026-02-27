---
estado: Specified
tipo: feature
---
# F04: Levels removal

**Epic**: [E10](../README.md)
**Satisface**: P5
**Objetivo**: Clean codebase with no dead `levels:` code
**Beneficio**: Eliminates maintenance burden of dual-path code and reduces cognitive overhead
**Milestone**: `grep -r "Levels" internal/` returns zero Go code matches; all tests pass

## Scope

**In**: Remove HierarchyLevel struct, ExpandLevels, CheckNesting, mergeLevels, stemhealth checks 8-9, v1 parsing path
**Out**: Any new feature work (this is pure cleanup)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Code cleanup](S001-code-cleanup/) | All levels-related code removed |
| S002 | [Final validation](S002-final-validation/) | Full regression confirms no breakage |

## Invariantes

- INV2 (heredado): Code coverage stays above 85%
- INV3 (heredado): MCP server behavior unchanged

## Dependencias

- F01, F02, F03 (remove old code only after everything uses new model)

## Fuente de verdad

- `internal/rules/rules.go` — HierarchyLevel struct
- `internal/rules/hierarchy.go` — ExpandLevels, CheckNesting
- `internal/rules/merge.go` — mergeLevels
- `internal/rules/stemhealth.go` — Checks 8-9
- `cmd/rootline/validate.go` — CheckNesting call sites
- `cmd/rootline/fix.go` — CheckNesting call site
