---
estado: Completed
tipo: task
---
# T005: Update docs tests and CLI contracts

**Outcome**: [O11 CLI output projection and shaping](README.md)
**Contribuye a**: CE1-CE4 del Outcome.

[[blocked_by:./T001-route-graph-json-through-shared-output.md]]
[[blocked_by:./T002-add-array-aware-field-extraction.md]]
[[blocked_by:./T003-add-query-select-projection-and-title.md]]
[[blocked_by:./T004-add-machine-output-formats-for-projections.md]]

## Preserva

- INV1: Existing unprojected JSON contracts remain backward compatible.
  - Verificar: contract docs distinguish base JSON from projected JSON.
- INV2: Rootline remains generic and does not encode roadmap-specific readiness or blocker policy.
  - Verificar: docs direct roadmap readiness use cases to roadmapctl where appropriate.
- INV3: Machine-readable output stays clean on stdout.
  - Verificar: docs state stdout/stderr discipline for machine modes.

## Contexto

The README currently states that all commands support `--output json|table` and `--field`, and includes examples like `rootline query --where 'estado == "Pending"' --field path`. After projection work, docs and tests must match actual behavior precisely.

## Alcance

**In**:
1. Update README and command docs for `--field`, array projections, query projections, and new output formats.
2. Correct stale or misleading examples.
3. Add regression tests for the exact no-Python workflows observed during roadmap discovery.
4. Update MCP or JSON-RPC docs only if CLI contracts affect shared result shapes.
5. Validate roadmap docs.

**Out**:
- Do not implement additional projection features beyond those delivered by T001-T004.
- Do not change roadmapctl docs except for cross-reference notes if needed.

## Estado inicial esperado

- T001-T004 have implemented graph shared output, array-aware field extraction, query projection, and selected machine formats.

## Criterios de Aceptación

- README examples are executable with current CLI behavior.
- Docs specify which commands support projection and how arrays behave.
- Tests cover replacing Python postprocessing for graph edges/broken links and query compact rows.
- `go test ./...` passes.
- `rootline validate --all docs/roadmap/` passes.

## Fuente de verdad

- `README.md`
- `docs/query.md`
- `docs/describe.md`
- `docs/json-rpc.md`
- `cmd/rootline/commands_test.go`
- `cmd/rootline/graph_test.go`
- `docs/roadmap/`
