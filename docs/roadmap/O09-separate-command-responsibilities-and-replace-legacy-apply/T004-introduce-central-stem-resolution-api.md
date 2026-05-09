---
estado: Specified
tipo: task
---
# T004: Introduce central stem resolution API

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE2 y CE4 del Outcome.

[[blocked_by:./T001-codify-command-responsibility-contracts.md]]

## Preserva

- INV3: Existing read-only commands remain read-only.
  - Verificar: resolver refactor does not add writes to read-only commands.

## Contexto

Multiple flows call `WalkUp`, `MergeStemFiles`, or `ResolveForRecord` differently. This causes inconsistent handling of match/scope/domain/provenance and contributed to `entries[0]` root-most targeting being mistaken for closest `.stem`.

## Alcance

**In**:
1. Add an internal resolver abstraction that returns the stem chain, effective schema, target metadata, and minimal provenance while preserving current v2 behavior.
2. Identify and migrate the highest-risk call sites behind the abstraction without changing observable behavior.
3. Add tests documenting root-to-leaf ordering and closest/root-most helper semantics.
4. Keep compatibility with existing `MergeStemFiles` callers during the transition.

**Out**:
- Enforcing monotonic constraints; tracked in O10.
- Changing roadmap `.stem` schema.
- Rewriting every caller in one pass if not needed for the new command boundaries.

## Estado inicial esperado

- `rules.WalkUp` returns root-to-leaf.
- `ResolveForRecord` applies match filtering, but many command paths do not use it.

## Criterios de Aceptación

- New resolver tests cover root-to-leaf chain, closest `.stem`, effective schema, and field source/provenance for nested stems.
- `apply`, `fix`, `analyze`, `describe`, or other touched command code no longer hand-rolls ambiguous closest/root-most logic.
- Existing tests for rules, validate, describe, and nested stems still pass.

## Fuente de verdad

- `internal/rules/discovery.go`
- `internal/rules/merge.go`
- `internal/rules/hierarchy.go`
- `internal/rules/match.go`
- `internal/rules/domains.go`
- `cmd/rootline/apply.go`
- `cmd/rootline/fix.go`
- `cmd/rootline/analyze.go`
