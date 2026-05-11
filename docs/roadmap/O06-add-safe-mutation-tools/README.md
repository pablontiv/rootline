---
estado: Pending
tipo: outcome
---
# O06: Add safe mutation tools

## Objetivo

Expose Rootline writes that are narrow, auditable, and validated after execution: creating records and setting frontmatter fields.

## Criterios de Éxito

- CE1: Pi can create and update Rootline records through guarded tools.
  - Verificar: Run tool-level tests for rootline_new and rootline_set.
- CE2: Every mutation validates affected files and reports failures clearly.
  - Verificar: Inspect test output and manual validation flow.

## Invariantes

- INV1: Mutating tools must not bypass Rootline validation or write outside user-approved paths.
  - Verificar: Review tool implementation and tests.

## Alcance

**In**:
- rootline_new tool
- rootline_set tool
- Confirmations/guards
- Post-write validation

**Out**:
- Bulk fix/migrate/apply workflows
- Publishing

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-design-mutation-guardrails.md) | Design guardrails for mutating Rootline tools. |
| [T002](T002-implement-rootline-new-tool.md) | Implement rootline_new for creating governed records. |
| [T003](T003-implement-rootline-set-tool.md) | Implement rootline_set for updating frontmatter fields. |
| [T004](T004-add-mutation-tests.md) | Add tests for success, validation failure, and blocked path cases. |
| [T005](T005-document-safe-mutation-workflows.md) | Document safe mutation workflows and non-goals. |
| [T006](T006-fix-section-aware-validation.md) | Make rootline validation paths see Markdown sections defined by .stem section fields, so section-aware mutations and validation agree. |
| [T007](T007-reconcile-set-create-contract.md) | Align rootline set --create implementation, help text, docs, and tests so agents know exactly whether it creates sections only or also scaffolds missing files. |
| [T008](T008-fix-new-scaffolding-defaults.md) | Prevent rootline new from blindly using the first enum value for discriminating fields such as roadmap tipo, and document the scaffold default policy. |
