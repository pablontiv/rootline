---
estado: Completed
tipo: feature
---
# F02: Automatic vertical consistency

**Epic**: [E10](../README.md)
**Satisface**: P2
**Objetivo**: Validation automatically warns when parent index files and children have inconsistent values for shared fields
**Beneficio**: Drift between parent and child records is caught without requiring `aggregate:` expressions — consistency comes from the structure
**Milestone**: `rootline validate` on a directory with an index `estado: In Progress` and all children `estado: Completed` produces a drift warning

## Scope

**In**: Drift comparison logic, drift warnings in validation output, match-scoped field exclusion
**Out**: Automatic fix/repair of drift (manual or via `aggregate:`), new `aggregate:` behavior

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Parent-child drift detection](S001-parent-child-drift-detection/) | Engine detects field drift between parent index and children |
| S002 | [Drift detection testing](S002-drift-detection-testing/) | Comprehensive test coverage for drift edge cases |

## Invariantes

- INV1 (heredado): All existing workflows produce identical results for v1 stems
- INV2 (heredado): Code coverage stays above 85%
- INV4: Drift warnings are warnings, not errors — they don't block validation

## Dependencias

- F01 (match-based field scoping must work first)

## Fuente de verdad

- `internal/rules/validate.go` — Validation engine
- `internal/derive/aggregate.go` — Existing aggregation (for reference, not modified)
- `docs/research/intrinsic-hierarchy-principle.md` — Part 3 (Automatic Vertical Consistency)
