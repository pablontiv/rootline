---
estado: In Progress
tipo: task
---
# T005: Adapt validation and query flows to layered resolver

**Outcome**: [O10 Move .stem to monotonic hierarchical constraints](README.md)
**Contribuye a**: CE3 del Outcome.

[[blocked_by:./T003-expose-stem-provenance-in-describe-explain.md]]
[[blocked_by:./T004-upgrade-stem-health-monotonic-diagnostics.md]]

## Preserva

- INV2: Effective schema output remains explainable to agents through provenance.
  - Verificar: validation errors and query/domain behavior can trace relevant schema sources.

## Contexto

Several flows bypass record-specific resolution or use merged schema inconsistently: analyze governance detectors use one root stem, derive/aggregate use default merge, directory describe/new expose unfiltered match fields, and domain aliases are not consistently wired into filters.

## Alcance

**In**:
1. Audit and migrate validation, query/filter, new, analyze, derive/aggregate, graph-related schema, and MCP callers to the layered resolver where behavior requires it.
2. Add tests for match-scoped fields, required-match, domain aliases, nested stems, and structural validation consistency.
3. Preserve current behavior for flows intentionally operating at directory/schema-summary level.
4. Document any flow that intentionally uses unfiltered directory-level schema.

**Out**:
- Changing command responsibility boundaries from O09.
- Adding destructive evolution syntax.

## Estado inicial esperado

- T003 and T004 expose provenance and health diagnostics.

## Criterios de Aceptación

- Validation, describe/explain, and query-related tests agree on effective schema for the same record.
- Match/scope/domain behavior is consistent or explicitly documented where different.
- MCP and CLI paths have parity for the affected operations.
- `go test ./internal/rules ./internal/query ./cmd/rootline ./internal/mcp` passes focused cases.

## Fuente de verdad

- `internal/rules/validate.go`
- `cmd/rootline/validate.go`
- `cmd/rootline/query.go`
- `cmd/rootline/new.go`
- `cmd/rootline/analyze.go`
- `internal/derive/pipeline.go`
- `internal/mcp/tools.go`
