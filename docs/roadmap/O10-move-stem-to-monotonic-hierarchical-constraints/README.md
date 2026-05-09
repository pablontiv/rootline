---
estado: Pending
tipo: outcome
---
# O10: Move .stem to monotonic hierarchical constraints

## Objetivo

Rootline evolves `.stem` from a YAML override cascade into a hierarchical constraint model where parent constraints apply downward, child stems add or narrow constraints, and destructive changes require explicit schema evolution.

## Criterios de Éxito

- CE1: The `.stem` constraint algebra defines what is additive, narrowing, widening, conflicting, or explicit evolution.
  - Verificar: design matrix covers type, required, enum values, severity, match/scope, defaults, domains, links, derive, aggregate, and structural rules.
- CE2: A layered resolver can expose effective constraints, provenance, and conflicts without losing v2 compatibility during migration.
  - Verificar: resolver tests show layer sources and current v2 behavior remains available.
- CE3: `describe`, `explain`, stem-health, validation, and query-related flows consume a consistent resolver model.
  - Verificar: CLI/MCP tests cover nested `.stem`, match-scoped fields, domain aliases, and conflicts.
- CE4: Destructive schema changes are represented as explicit schema evolution, not ordinary child overrides.
  - Verificar: tests reject child loosening/removal/replacement under monotonic mode unless an evolution operation exists.
- CE5: Docs and roadmap schema reflect the new model, including valid child narrowing such as string-to-enum where approved.
  - Verificar: docs and roadmap validation pass with accepted warnings removed or justified.

## Invariantes

- INV1: Moving down the directory tree must never silently reduce parent guarantees.
  - Verificar: monotonic health diagnostics fail loosening/widening/removal without explicit evolution.
- INV2: Effective schema output remains explainable to agents through provenance.
  - Verificar: `describe`/`explain` JSON includes source layers and conflicts.
- INV3: v2 compatibility or migration behavior is explicit.
  - Verificar: tests distinguish legacy v2 cascade from monotonic mode or migration path.

## Alcance

**In**:
- Constraint algebra and compatibility design.
- Layered resolver and provenance/conflict representation.
- Describe/explain output updates.
- Stem-health diagnostics for narrowing versus destructive override.
- Validation/query/new/analyze/fix resolver alignment where needed.
- Explicit schema evolution for destructive changes.
- Docs/tests/schema cleanup.

**Out**:
- Initial command responsibility split; tracked in O09.
- Pi extension UX.
- Marketplace publishing.

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-design-monotonic-stem-constraint-algebra.md) | Design the monotonic `.stem` constraint algebra. |
| [T002](T002-implement-layered-stem-resolver.md) | Implement a layered `.stem` resolver with compatibility mode. |
| [T003](T003-expose-stem-provenance-in-describe-explain.md) | Expose stem provenance and conflicts in describe/explain. |
| [T004](T004-upgrade-stem-health-monotonic-diagnostics.md) | Upgrade stem-health diagnostics for monotonic constraints. |
| [T005](T005-adapt-validation-and-query-flows-to-layered-resolver.md) | Adapt validation and query-related flows to layered resolution. |
| [T006](T006-add-schema-evolution-for-destructive-changes.md) | Add explicit schema evolution for destructive changes. |
| [T007](T007-update-stem-docs-tests-and-roadmap-schema.md) | Update `.stem` docs, tests, and roadmap schema. |
