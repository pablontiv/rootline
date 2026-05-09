---
estado: Specified
tipo: task
---
# T009: Implement repair apply data-only

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE3 del Outcome.

[[blocked_by:./T005-normalize-proposal-taxonomy.md]]
[[blocked_by:./T003-neutralize-legacy-apply-safety-risks.md]]

## Preserva

- INV1: `.stem` mutation is explicit schema evolution, never a hidden side effect of data repair.
  - Verificar: repair apply tests compare `.stem` bytes before and after.

## Contexto

Rootline needs a bulk data repair path that can correct frontmatter/body/link data against an existing schema without changing governance. Existing safe patterns include `set` rollback and parts of `fix` proposal application.

## Alcance

**In**:
1. Add a data-only repair apply engine or command that accepts repair proposals.
2. Support dry-run, path containment, post-validation, and rollback for modified Markdown files.
3. Reuse safe pieces from `set` and `fix` where appropriate.
4. Reject schema-surface proposals such as `extend_enum`, `add_aggregate`, and `remove_stem_field`.
5. Emit versioned JSON result with changed files, skipped items, dry-run status, and validation summary.

**Out**:
- Schema proposal/application.
- Guessing schema changes from invalid data values.

## Estado inicial esperado

- T005 has classified proposal surfaces.
- T003 has made legacy dry-run behavior safer while replacement work proceeds.

## Criterios de Aceptación

- Repair apply modifies Markdown/frontmatter only.
- Repair apply never creates, deletes, or modifies `.stem` files.
- Dry-run writes nothing.
- Failed post-validation rolls back modified files or reports chosen non-transactional contract explicitly.
- Tests cover `correct_value`, `migrate_value`, `add_field`, section/body repairs where supported, and rejection of schema proposals.

## Fuente de verdad

- `cmd/rootline/set.go`
- `cmd/rootline/fix.go`
- `internal/fix/fix.go`
- `internal/proposal/proposal.go`
- `internal/infer/apply.go`
