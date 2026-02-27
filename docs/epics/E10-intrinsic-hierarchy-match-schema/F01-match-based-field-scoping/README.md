---
estado: Completed
tipo: feature
---
# F01: Match-based field scoping

**Epic**: [E10](../README.md)
**Satisface**: P1
**Objetivo**: Fields in `.stem` schema can be scoped to specific directory patterns via `match:` on the field itself, eliminating `levels:`
**Beneficio**: Removes redundant hierarchy declaration — the filesystem IS the hierarchy
**Milestone**: `rootline validate` resolves per-level fields using `SchemaField.Match` instead of `ExpandLevels`

## Scope

**In**: SchemaField.Match field, v2 stem parsing, match-based resolution replacing ExpandLevels
**Out**: Automatic drift detection (F02), tooling updates (F03), levels removal (F04)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [SchemaField match extension](S001-schemafield-match-extension/) | SchemaField carries Match and v2 stems parse correctly |
| S002 | [Replace ExpandLevels with match resolution](S002-replace-expandlevels-with-match/) | ResolveForRecord uses match filtering instead of virtual stems |

## Invariantes

- INV1 (heredado): All existing workflows produce identical results for v1 stems
- INV2 (heredado): Code coverage stays above 85%

## Dependencias

- None (foundation feature)

## Fuente de verdad

- `internal/rules/rules.go` — SchemaField struct
- `internal/rules/hierarchy.go` — ExpandLevels, ResolveForRecord
- `internal/rules/merge.go` — mergeLevels
- `docs/research/intrinsic-hierarchy-principle.md` — Part 3 (.stem v2 format)
