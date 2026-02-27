---
estado: Completed
tipo: feature
---
# F03: Tooling adaptation

**Epic**: [E10](../README.md)
**Satisface**: P3, P4
**Objetivo**: All .stem-generating tools produce v2 match-based schemas
**Beneficio**: Users creating new projects or migrating existing ones get the new format automatically
**Milestone**: `rootline init` on a hierarchical directory produces a v2 `.stem` with `match:` fields; `rootline migrate --from-levels` converts v1 to v2

## Scope

**In**: Schema inference output, init command, migrate --from-levels command, project stem migration
**Out**: New inference heuristics, new match pattern types

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Schema inference update](S001-schema-inference-update/) | Infer and init produce v2 match-based schemas |
| S002 | [Migration from levels](S002-migration-from-levels/) | Automated v1 → v2 .stem conversion |

## Invariantes

- INV1 (heredado): All existing workflows produce identical results for v1 stems
- INV2 (heredado): Code coverage stays above 85%
- INV5: Migration preserves field semantics exactly — no behavioral change

## Dependencias

- F01 (match-based field scoping must work for generated stems to be usable)

## Fuente de verdad

- `internal/infer/hierarchy.go` — ToLevelsMap, distributeFields
- `cmd/rootline/init.go` — buildHierarchicalStems, generateHierarchicalRootYAML
- `internal/migrate/` — Migration engine
- `docs/epics/.stem` — Only levels: stem in the project
