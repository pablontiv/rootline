---
estado: Specified
tipo: task
---
# T003: Expose stem provenance in describe and explain

**Outcome**: [O10 Move .stem to monotonic hierarchical constraints](README.md)
**Contribuye a**: CE2 y CE3 del Outcome.

[[blocked_by:./T002-implement-layered-stem-resolver.md]]

## Preserva

- INV2: Effective schema output remains explainable to agents through provenance.
  - Verificar: `describe` and `explain` JSON include layers/sources/conflicts.

## Contexto

`describe` and `explain` are the main introspection tools for agents and users. Current outputs flatten schema fields and show only one source path per field. A monotonic architecture requires showing how parent and child constraints compose.

## Alcance

**In**:
1. Extend describe/explain result models with constraint layers, sources, narrowed-by data, and conflicts.
2. Preserve existing flat schema output where needed for compatibility.
3. Fix file-target explain/describe paths to use record-specific resolution where appropriate.
4. Add JSON tests for nested `.stem`, match-scoped fields, and conflict display.

**Out**:
- Changing validation behavior.
- Applying schema evolution.

## Estado inicial esperado

- T002 resolver exposes layered provenance/conflicts.

## Criterios de Aceptación

- `rootline describe` can show both effective schema and layer provenance.
- `rootline explain <file>` reflects record-specific match filtering and sources.
- Existing consumers of flat fields are not broken unless an explicit version bump is approved.
- Docs describe the new introspection fields.

## Fuente de verdad

- `cmd/rootline/describe.go`
- `cmd/rootline/explain.go`
- `internal/rules/describe.go`
- `internal/rules/explain.go`
- `docs/describe.md`
