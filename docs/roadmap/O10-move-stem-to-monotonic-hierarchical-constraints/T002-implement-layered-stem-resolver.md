---
estado: Completed
tipo: task
---
# T002: Implement layered stem resolver

**Outcome**: [O10 Move .stem to monotonic hierarchical constraints](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T001-design-monotonic-stem-constraint-algebra.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T004-introduce-central-stem-resolution-api.md]]

## Preserva

- INV2: Effective schema output remains explainable to agents through provenance.
  - Verificar: resolver result includes source layers for effective constraints.
- INV3: v2 compatibility or migration behavior is explicit.
  - Verificar: tests cover legacy and monotonic resolution modes according to T001.

## Contexto

The central resolver from O09 is the bridge for changing semantics safely. This task implements layered constraint resolution that can retain provenance and conflicts rather than flattening everything into one last-writer field source.

## Alcance

**In**:
1. Implement resolver result types for stem chain, layer constraints, effective constraints, provenance, and conflicts.
2. Preserve current v2 cascade behavior where compatibility requires it.
3. Add monotonic resolution mode according to T001.
4. Add tests for nested `.stem` chains, same-name field layers, match-scoped fields, domains, links, and structural rules.
5. Add caching strategy or note performance boundaries for per-record resolution.

**Out**:
- Updating all CLI outputs.
- Schema evolution syntax.

## Estado inicial esperado

- O09/T004 introduced a central resolver API.
- T001 defines the algebra and compatibility strategy.

## Criterios de Aceptación

- Layered resolver exposes effective constraints plus provenance and conflicts.
- Legacy v2 tests continue to pass or are explicitly moved under legacy-mode tests.
- Monotonic-mode tests cover add, narrow, conflict, and destructive operation detection.
- No read-only command gains write behavior.

## Fuente de verdad

- `internal/rules/discovery.go`
- `internal/rules/merge.go`
- `internal/rules/hierarchy.go`
- `internal/rules/match.go`
- `internal/rules/domains.go`
- `internal/rules/stem_cache.go`
