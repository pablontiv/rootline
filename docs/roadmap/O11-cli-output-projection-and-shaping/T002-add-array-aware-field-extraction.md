---
estado: Completed
tipo: task
---
# T002: Add array-aware field extraction

**Outcome**: [O11 CLI output projection and shaping](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T001-route-graph-json-through-shared-output.md]]

## Preserva

- INV1: Existing unprojected JSON contracts remain backward compatible.
  - Verificar: commands without `--field` return the same JSON contracts.
- INV3: Machine-readable output stays clean on stdout.
  - Verificar: extraction errors return normal CLI errors, not partial JSON.

## Contexto

`--field` is repeatable in the global flags, but shared extraction currently uses only the first value and navigates only object dot paths. This is insufficient for Rootline result shapes such as `query` rows and graph edges, which are arrays under `rows[]` and `edges[]`.

## Alcance

**In**:
1. Extend field extraction to understand array projection syntax for common paths, e.g. `rows[].path`, `rows[].frontmatter.estado`, `edges[].source`.
2. Keep simple object paths such as `summary` and `schema.estado.values` working.
3. Define behavior for missing keys inside projected arrays.
4. Add focused tests for object paths, array projections, missing keys, and invalid traversal.

**Out**:
- Do not implement a full jq/JMESPath language.
- Do not change query or graph base result structs.

## Estado inicial esperado

- `extractField` unmarshals JSON and traverses only `map[string]any` by dot-separated path.
- Multiple `--field` values are accepted by flags but only `fieldPath[0]` is used.

## Criterios de Aceptación

- `--field rows[].path` returns a JSON array of paths for query results.
- `--field rows[].frontmatter.estado` returns statuses for query results.
- `--field edges[].source` returns graph edge sources.
- Existing `--field summary` and `--field schema.estado` behavior remains valid.
- `go test ./cmd/rootline` passes.

## Fuente de verdad

- `cmd/rootline/root.go`
- `cmd/rootline/validate.go`
- `cmd/rootline/commands_test.go`
- `README.md`
- `docs/describe.md`
