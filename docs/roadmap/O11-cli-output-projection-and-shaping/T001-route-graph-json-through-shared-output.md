---
estado: In Progress
tipo: task
---
# T001: Route graph JSON through shared output handling

**Outcome**: [O11 CLI output projection and shaping](README.md)
**Contribuye a**: CE1 del Outcome.

## Preserva

- INV1: Existing unprojected JSON contracts remain backward compatible.
  - Verificar: existing graph JSON tests keep passing.
- INV3: Machine-readable output stays clean on stdout.
  - Verificar: JSON output remains parseable and diagnostics stay on stderr.

## Contexto

`rootline graph --output json` currently marshals and writes its JSON result directly in `cmd/rootline/graph.go`, bypassing shared `outputJSON`. Because of that, global `--field` extraction is ignored for graph even though Rootline documentation says commands support `--field`.

## Alcance

**In**:
1. Change graph JSON mode to build the same `GraphResult` but emit it through shared output handling.
2. Preserve existing graph JSON shape: `version`, `kind`, `nodes`, `edges`, `cycles`, `broken_links`.
3. Add or update tests proving `graph --output json --field edges` works and unprojected graph JSON remains unchanged.

**Out**:
- Do not change graph edge resolution semantics.
- Do not add roadmap-specific graph analysis.

## Estado inicial esperado

- `cmd/rootline/graph.go` writes JSON directly with `json.Marshal` and `fmt.Fprintln`.
- `outputJSON` lives in `cmd/rootline/validate.go` and handles `--field` for commands that call it.

## Criterios de Aceptación

- `rootline graph <dir> --output json --field edges` outputs a JSON array of edges.
- `rootline graph <dir> --output json --field broken_links` outputs a JSON array.
- Existing `rootline graph <dir> --output json` contract remains compatible.
- `go test ./cmd/rootline ./internal/graph` passes.

## Fuente de verdad

- `cmd/rootline/graph.go`
- `cmd/rootline/validate.go`
- `cmd/rootline/graph_test.go`
- `internal/graph/graph.go`
