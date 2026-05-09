---
estado: Specified
tipo: task
---
# T010: Make fix all schema-safe by default

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE3 del Outcome.

[[blocked_by:./T009-implement-repair-apply-data-only.md]]

## Preserva

- INV1: `.stem` mutation is explicit schema evolution, never a hidden side effect of data repair.
  - Verificar: `fix --all` default apply tests leave `.stem` bytes unchanged.

## Contexto

Investigation confirmed `fix --all` can mutate `.stem` through `extend_enum`, `add_aggregate`, and `remove_stem_field`. Under the new responsibility model, those must become schema suggestions or schema proposals, not default repair actions.

## Alcance

**In**:
1. Split or filter `fix --all` proposals so schema-surface proposals are not applied by default.
2. Preserve dry-run visibility of schema suggestions with clear handoff to schema propose/apply.
3. Update `fix.ApplyProposals` or caller logic to enforce data-only behavior by default.
4. Add tests for existing schema-mutating proposal types.

**Out**:
- Removing schema suggestions entirely.
- Implementing all schema proposal operations if T008 already covers them.

## Estado inicial esperado

- T009 provides a data-only repair apply path or engine.

## Criterios de Aceptación

- `rootline fix --all` no longer mutates `.stem` unless an explicit schema-evolution mode is added and approved.
- Dry-run output distinguishes repair proposals from schema suggestions.
- Tests cover `extend_enum`, `add_aggregate`, and `remove_stem_field` behavior.
- Docs for `fix` state the new boundary clearly.

## Fuente de verdad

- `cmd/rootline/fix.go`
- `internal/fix/fix.go`
- `internal/proposal/proposal.go`
- `internal/proposal/stem_health.go`
- `docs/fix.md`
