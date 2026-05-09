---
estado: Specified
tipo: task
---
# T001: Codify command responsibility contracts

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE1, CE2, CE3 y CE4 del Outcome.

## Preserva

- INV1: `.stem` mutation is explicit schema evolution, never a hidden side effect of data repair.
  - Verificar: the contract explicitly classifies every mutating command by target surface.
- INV3: Existing read-only commands remain read-only.
  - Verificar: read-only commands are listed and have no write responsibilities.

## Contexto

The investigation found that `rootline apply` currently mixes bootstrap (`missing_schema` scaffold), schema mutation, and data correction. `fix --all` also mixes document repairs with schema mutations such as `extend_enum`, `add_aggregate`, and `remove_stem_field`. Before implementation, Rootline needs an explicit command responsibility contract.

## Alcance

**In**:
1. Define command lanes: discovery/read-only, governance validation, data repair, schema proposal, schema apply/evolution, and direct single-record mutation.
2. Classify current commands: `analyze`, `apply`, `fix`, `set`, `migrate`, `init`, `new`, `validate`, `describe`, `query`, `tree`, `stats`, `graph`, `explain`, MCP tools.
3. Define which commands may mutate Markdown, which may mutate `.stem`, and which must remain read-only.
4. Decide deprecation strategy for generic mixed `apply`.

**Out**:
- Implementing new commands.
- Changing `.stem` merge semantics.

## Estado inicial esperado

- Investigation artifacts exist in prior Pi session outputs for `apply` bugs, command API surface, data-first bootstrap, and `.stem` architecture.
- Existing O07 excludes core command bugfixes, so this task creates the core contract O07 can depend on.

## Criterios de Aceptación

- A checked-in ADR/spec or roadmap design document states the command responsibility contract.
- The spec explicitly says normal data repair must not mutate `.stem`.
- The spec identifies the replacement shape for legacy `apply`: schema proposal/apply and repair apply.
- The spec lists compatibility/deprecation expectations for existing CLI and MCP consumers.
- `rootline validate --all docs/roadmap/` returns exit 0 or only accepted warnings.

## Fuente de verdad

- `cmd/rootline/apply.go`
- `cmd/rootline/fix.go`
- `cmd/rootline/migrate.go`
- `cmd/rootline/set.go`
- `cmd/rootline/analyze.go`
- `internal/infer/report.go`
- `internal/proposal/proposal.go`
- `internal/mcp/tools.go`
- `docs/roadmap/O07-expose-complex-operations-with-guardrails/README.md`
