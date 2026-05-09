---
estado: Pending
tipo: outcome
---
# O11: CLI output projection and shaping

## Objetivo

Rootline can return compact, machine-friendly projections for query, graph, and validation data so agents and shell workflows do not need ad-hoc Python postprocessing for common inspection tasks.

## Criterios de Éxito

- CE1: JSON field extraction is consistent across commands that advertise `--field`.
  - Verificar: `rootline graph docs/roadmap/ --output json --field edges` returns only graph edges as valid JSON.
- CE2: `--field` can project through arrays for common Rootline result shapes.
  - Verificar: tests cover paths such as `rows[].path`, `rows[].frontmatter.estado`, and `edges[].source`.
- CE3: `query` can emit compact row projections without full Markdown bodies when requested.
  - Verificar: a query can output `path`, `estado`, computed `title`, and `links` without requiring a Python script.
- CE4: machine output formats and docs clearly state stable contracts and stdout/stderr behavior.
  - Verificar: docs and tests cover JSON projection behavior, table behavior, and any added `jsonl`, `csv`, or `tsv` modes.

## Invariantes

- INV1: Existing unprojected JSON contracts remain backward compatible.
  - Verificar: existing query, graph, validate, and MCP tests keep passing.
- INV2: Rootline remains generic and does not encode roadmap-specific readiness or blocker policy.
  - Verificar: no roadmap-specific commands or status semantics are added to Rootline for this outcome.
- INV3: Machine-readable output stays clean on stdout.
  - Verificar: progress, warnings, and diagnostics that are not part of the JSON contract go to stderr.

## Alcance

**In**:
- Shared JSON output path for graph.
- Array-aware `--field` extraction.
- Query projection or select support for compact records.
- Computed title extraction for query projections if accepted as a generic record summary.
- Optional machine output formats for projected rows.
- Docs and tests for CLI contracts.

**Out**:
- Roadmap-specific `ready`, `blocked`, or scoring logic; that belongs in `roadmapctl`.
- Replacing `jq`, JMESPath, or full transformation languages in the first iteration.
- MCP mutation or Pi extension work.

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-route-graph-json-through-shared-output.md) | Route graph JSON through shared output handling. |
| [T002](T002-add-array-aware-field-extraction.md) | Add array-aware field extraction for JSON results. |
| [T003](T003-add-query-select-projection-and-title.md) | Add compact query projections, including title. |
| [T004](T004-add-machine-output-formats-for-projections.md) | Add or design machine formats for projected rows. |
| [T005](T005-update-docs-tests-and-contracts.md) | Update docs, tests, and CLI contracts. |
