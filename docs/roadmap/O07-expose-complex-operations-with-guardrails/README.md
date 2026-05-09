---
estado: Pending
tipo: outcome
---
# O07: Expose complex operations with guardrails

## Objetivo

Decide and implement protected interfaces for Rootline operations that can affect many files or schema state, while preserving user control and validation.

## Criterios de Éxito

- CE1: Complex operations have explicit approval and validation workflows.
  - Verificar: Review command/tool designs and tests.
- CE2: The extension exposes only complex operations that have safe UX and clear rollback guidance.
  - Verificar: Run manual dry-run/preview style workflows where applicable.

## Invariantes

- INV1: Bulk operations require explicit user intent and cannot be triggered silently by agent context guidance.
  - Verificar: Inspect commands, prompts, and tool activation defaults.

## Alcance

**In**:
- rootline_fix workflow
- rootline_migrate workflow
- rootline_analyze/apply protected workflow
- Rollback/preview documentation

**Out**:
- Fixing Rootline core command bugs
- Publishing package

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-design-complex-operation-ux.md) | Design UX for complex Rootline operations. |
| [T002](T002-implement-protected-fix-workflow.md) | Implement protected workflow for rootline fix. |
| [T003](T003-implement-protected-migrate-workflow.md) | Implement protected workflow for rootline migrate. |
| [T004](T004-implement-analyze-apply-workflow.md) | Implement protected workflow around analyze reports and apply. |
| [T005](T005-document-complex-operation-risks.md) | Document complex operation risk model and rollback procedure. |
