---
estado: Completed
tipo: task
---
# T003: Add compact query projections including title

**Outcome**: [O11 CLI output projection and shaping](README.md)
**Contribuye a**: CE3 del Outcome.

[[blocked_by:./T002-add-array-aware-field-extraction.md]]

## Preserva

- INV1: Existing unprojected JSON contracts remain backward compatible.
  - Verificar: `rootline query ... --output json` without projection still returns full records.
- INV2: Rootline remains generic and does not encode roadmap-specific readiness or blocker policy.
  - Verificar: projection fields are record fields or generic computed summaries only.

## Contexto

The observed postprocessing script reopened `rootline query` JSON, extracted `path`, `estado`, a title from the first Markdown heading, and `links`, while discarding full `body`. Rootline should provide a generic compact projection mode for this shape instead of requiring Python.

## Alcance

**In**:
1. Add a query projection flag such as `--select path,estado,title,links` or an equivalent approved syntax.
2. Resolve selected names from record path, effective frontmatter/derived fields, links, and generic computed fields.
3. Implement `title` as a generic record summary from the first Markdown heading or first non-empty body line.
4. Ensure projected JSON omits full `body` unless explicitly selected.
5. Add table behavior or document JSON-only projection behavior.

**Out**:
- Do not add roadmap-specific fields like `ready`, `blocked`, or `blocking_dependencies`.
- Do not embed a full expression language for projection in this task.

## Estado inicial esperado

- `query` returns `rows` containing full `body`, `frontmatter`, `links`, and `derived`.
- `renderQueryTable` auto-discovers all frontmatter and derived keys, which can be too broad for agent-readable summaries.

## Criterios de Aceptación

- A command can output compact rows containing `path`, `estado`, `title`, and `links` without Python.
- Projected query JSON remains versioned and parseable.
- Unprojected query JSON remains backward compatible.
- Tests cover frontmatter field selection, derived field selection, `path`, `links`, missing fields, and `title` extraction.
- `go test ./cmd/rootline ./internal/query` passes.

## Fuente de verdad

- `cmd/rootline/query.go`
- `internal/query/expr_eval.go`
- `internal/extract/extract.go`
- `cmd/rootline/commands_test.go`
- `docs/query.md`
