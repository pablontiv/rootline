---
estado: Completed
tipo: task
---
# T006: Add schema evolution for destructive changes

**Outcome**: [O10 Move .stem to monotonic hierarchical constraints](README.md)
**Contribuye a**: CE4 del Outcome.

[[blocked_by:./T004-upgrade-stem-health-monotonic-diagnostics.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T008-implement-schema-apply-explicit.md]]

## Preserva

- INV1: Moving down the directory tree must never silently reduce parent guarantees.
  - Verificar: destructive changes require explicit evolution operations.

## Contexto

Some destructive changes are legitimate during migrations: remove a field, rename a field, replace enum values, loosen required, or change meaning. These should not happen through ordinary child `.stem` shadowing; they need explicit schema evolution with classification and validation.

## Alcance

**In**:
1. Define representation for destructive schema evolution operations in proposals, migration plans, or `.stem` metadata.
2. Integrate with schema apply from O09 where appropriate.
3. Reuse or adapt `migrate` diff/breaking-change classification.
4. Add tests for remove, rename, replace enum, loosen required, severity loosen, and incompatible type change.
5. Ensure evolution operations produce clear validation/rollback guidance.

**Out**:
- Data repair application.
- Arbitrary automatic schema mutation from data drift.

## Estado inicial esperado

- T004 can detect destructive child operations.
- O09/T008 provides explicit schema apply infrastructure.

## Criterios de Aceptación

- Destructive changes are rejected as normal child constraints under monotonic mode.
- The same changes can be represented as explicit schema evolution where approved.
- Migration/schema apply tests classify breaking versus safe changes.
- Docs explain when to use schema evolution instead of child narrowing.

## Fuente de verdad

- `internal/migrate/diff.go`
- `internal/migrate/rename.go`
- `cmd/rootline/migrate.go`
- `internal/infer/apply.go`
- `internal/proposal/proposal.go`
