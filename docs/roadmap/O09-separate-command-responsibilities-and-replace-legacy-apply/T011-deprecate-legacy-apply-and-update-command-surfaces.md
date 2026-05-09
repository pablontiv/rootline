---
estado: In Progress
tipo: task
---
# T011: Deprecate legacy apply and update command surfaces

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE1, CE2, CE3 y CE4 del Outcome.

[[blocked_by:./T008-implement-schema-apply-explicit.md]]
[[blocked_by:./T009-implement-repair-apply-data-only.md]]
[[blocked_by:./T010-make-fix-all-schema-safe-by-default.md]]

## Preserva

- INV2: Every machine-readable command used by agents emits parseable, versioned JSON.
  - Verificar: replacement commands and any legacy shim have tested JSON contracts.

## Contexto

Once schema apply and repair apply exist, generic `rootline apply` should no longer be the primary workflow. The repo is pre-1.0 and has limited legacy constraints, so the command can be deprecated, narrowed, or removed according to the contract from T001.

## Alcance

**In**:
1. Implement the approved deprecation/narrowing strategy for `rootline apply`.
2. Ensure legacy `apply` no longer performs unsafe mixed writes silently.
3. Update command help, completion behavior if needed, and tests.
4. Decide how `analyze` reports relate to replacement proposal commands.
5. Add compatibility notes for scripts/agents.

**Out**:
- Pi extension workflow implementation.
- Monotonic `.stem` semantics.

## Estado inicial esperado

- T008 and T009 provide replacement write paths.
- T010 has made fix default behavior schema-safe.

## Criterios de Aceptación

- `rootline apply --help` clearly states deprecated/narrowed behavior or is removed according to the approved plan.
- Machine-readable consumers have a documented replacement path.
- Tests cover wrong report kind/version and legacy command behavior.
- O07 tasks can depend on the new safe surfaces instead of generic `apply`.

## Fuente de verdad

- `cmd/rootline/apply.go`
- `cmd/rootline/analyze.go`
- `cmd/rootline/root.go`
- `docs/roadmap/O07-expose-complex-operations-with-guardrails/T004-implement-analyze-apply-workflow.md`
