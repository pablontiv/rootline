---
estado: Completed
tipo: feature
---
# E10: Intrinsic Hierarchy — Match-Based Schema

**Metrica de exito**: `rootline validate` resolves per-level fields via `SchemaField.Match` without `levels:`, and automatically detects parent-child drift
**Timeline**: 2026-Q1 — en curso

## Intencion

Replace the explicit `levels:` hierarchy declaration in `.stem` files with a `match:`-based field scoping model on `SchemaField` itself. The filesystem already defines parent-child relationships — `levels:` is redundant. Additionally, add automatic vertical consistency detection so that drift between parent and child records is caught without requiring `aggregate:` expressions.

Based on research: `docs/research/intrinsic-hierarchy-principle.md`

## Postcondiciones

- P1: `.stem` v2 format works without `levels:` section — fields use `match:` for per-pattern scoping
- P2: Validation automatically detects parent-child field drift without `aggregate:`
- P3: `rootline init` and `rootline infer` generate v2 match-based schemas; `rootline migrate --from-levels` converts v1 to v2

## Invariantes

- INV1: All existing `rootline validate` / `rootline fix` / `rootline query` workflows produce identical results for v1 stems during transition
- INV2: Code coverage stays above 85%
- INV3: MCP server behavior unchanged (abstracts through engine)

## Out of Scope

- Conditional `validate:` rules with `if: {match: "E*"}` (deferred)
- Depth-based selectors or path patterns (glob-only for now)
- Partial exclusion beyond `.stemignore`

## Features

| ID | Nombre | Descripcion |
|----|--------|-------------|
| F01 | [Match-based field scoping](F01-match-based-field-scoping/) | SchemaField gets Match, replaces ExpandLevels |
| F02 | [Automatic vertical consistency](F02-automatic-vertical-consistency/) | Drift detection without aggregate |
| F03 | [Tooling adaptation](F03-tooling-adaptation/) | Update infer/init/migrate to produce v2 format |
| F04 | [Levels removal](F04-levels-removal/) | Remove all dead levels code |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | Foundation: new resolution model |
| F02 | F01 | Drift detection needs match-based resolution working |
| F03 | F01 | Tooling must generate the new format |
| F04 | F01, F02, F03 | Remove old code only after everything uses new model |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-02-27 | Glob-only match patterns (no depth/path selectors) | Simplest; consistent with existing filepath.Match |
| 2026-02-27 | Migration via `rootline migrate --from-levels` (option c) | Most aligned with existing migrate tooling |
| 2026-02-27 | Sequence id uses match map syntax for per-level prefix/digits | New projects need explicit config |

## Gaps Activos

- None identified
