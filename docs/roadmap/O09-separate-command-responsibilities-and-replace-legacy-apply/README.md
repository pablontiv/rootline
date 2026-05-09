---
estado: Pending
tipo: outcome
---
# O09: Separate command responsibilities and replace legacy apply

## Objetivo

Rootline separates read-only discovery, data repair, schema proposal, and explicit schema evolution so bulk commands no longer mix `.stem` mutation, document repair, and bootstrap side effects behind generic `apply`.

## Criterios de Éxito

- CE1: Current `apply` safety bugs are characterized and either neutralized or deprecated behind safer commands.
  - Verificar: focused CLI tests cover dry-run no-write, JSON purity, target selection, and partial-write behavior.
- CE2: Rootline has explicit, versioned proposal contracts for schema changes and data repairs.
  - Verificar: inspect JSON `kind/version`, operation surface, target paths, and requires-agent metadata in tests.
- CE3: Data repair flows cannot mutate `.stem` files by default.
  - Verificar: tests prove `repair apply` and default `fix --all` leave `.stem` bytes unchanged.
- CE4: Schema bootstrap/evolution flows mutate only `.stem` files through explicit schema commands with true dry-run and post-validation.
  - Verificar: tests prove schema dry-run writes nothing, schema apply targets explicit stems, and documents are not modified.
- CE5: Public docs, skills, MCP catalog, and roadmap dependencies no longer recommend unsafe mixed `apply` workflows.
  - Verificar: grep/docs review plus `rootline validate --all docs/roadmap/`.

## Invariantes

- INV1: `.stem` mutation is explicit schema evolution, never a hidden side effect of data repair.
  - Verificar: tests assert data repair commands do not touch `.stem`.
- INV2: Every machine-readable command used by agents emits parseable, versioned JSON.
  - Verificar: CLI tests JSON-parse stdout for affected commands.
- INV3: Existing read-only commands remain read-only.
  - Verificar: no new writes in `analyze`, `validate`, `describe`, `query`, `tree`, `stats`, `graph`, or `explain`.

## Alcance

**In**:
- Characterization tests for current `apply`/`fix --all` safety issues.
- Command responsibility contract for discovery, repair, schema propose, and schema apply.
- Central `.stem` resolution API needed by the new command boundaries.
- Proposal taxonomy shared by analyze/fix/schema/repair workflows.
- Data-first schema bootstrap via explicit read-only schema proposals.
- Explicit schema apply and data-only repair apply engines.
- Deprecation or narrowing of legacy `apply`.
- Docs/skills/MCP/CI cleanup related to these contracts.

**Out**:
- Full monotonic `.stem` semantics rollout; tracked in O10.
- Pi extension/package implementation; existing O03-O08 track Pi-facing work.
- Marketplace publishing.

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-codify-command-responsibility-contracts.md) | Codify command responsibility contracts and migration boundaries. |
| [T002](T002-add-apply-safety-characterization-tests.md) | Add failing tests for known `apply` safety issues. |
| [T003](T003-neutralize-legacy-apply-safety-risks.md) | Neutralize immediate legacy `apply` dry-run and JSON risks. |
| [T004](T004-introduce-central-stem-resolution-api.md) | Introduce a central `.stem` resolution API. |
| [T005](T005-normalize-proposal-taxonomy.md) | Normalize proposal taxonomy across analyze, fix, repair, and schema flows. |
| [T006](T006-extract-schema-generation-services-from-init.md) | Extract reusable schema generation services from init/hierarchy/split code. |
| [T007](T007-implement-schema-propose-bootstrap-and-incremental.md) | Implement read-only schema proposal generation for bootstrap and incremental use. |
| [T008](T008-implement-schema-apply-explicit.md) | Implement explicit schema apply with true dry-run and validation. |
| [T009](T009-implement-repair-apply-data-only.md) | Implement data-only repair apply. |
| [T010](T010-make-fix-all-schema-safe-by-default.md) | Make `fix --all` schema-safe by default. |
| [T011](T011-deprecate-legacy-apply-and-update-command-surfaces.md) | Deprecate or narrow legacy `apply` and update command surfaces. |
| [T012](T012-clean-up-command-docs-mcp-ci-and-skill-contracts.md) | Clean up docs, MCP catalog, CI recipes, and skill contracts. |
