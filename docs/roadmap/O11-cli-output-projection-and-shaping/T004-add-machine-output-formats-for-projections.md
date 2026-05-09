---
estado: Specified
tipo: task
---
# T004: Add machine output formats for projected rows

**Outcome**: [O11 CLI output projection and shaping](README.md)
**Contribuye a**: CE4 del Outcome.

[[blocked_by:./T003-add-query-select-projection-and-title.md]]

## Preserva

- INV1: Existing unprojected JSON contracts remain backward compatible.
  - Verificar: existing `--output json|table` behavior remains available.
- INV3: Machine-readable output stays clean on stdout.
  - Verificar: CSV/TSV/JSONL outputs contain only records on stdout.

## Contexto

External CLI precedent shows that JSON projection often pairs with stream or tabular machine formats such as JSONL, CSV, and TSV. Rootline should decide and implement the minimal output formats that make projected rows useful in shell pipelines without forcing Python.

## Alcance

**In**:
1. Decide the initial supported format set for projected query results: likely `jsonl`, `csv`, and/or `tsv`.
2. Implement deterministic column ordering from projection order.
3. Define escaping, headers, and `--no-header` behavior if CSV/TSV is included.
4. Add tests for projected rows in each accepted format.

**Out**:
- Do not implement streaming scan internals unless required by the selected format.
- Do not add format support for commands where result shape is not row-oriented unless explicitly justified.

## Estado inicial esperado

- Global output format is documented as `json|table`.
- Query table rendering is human-oriented and auto-discovers fields.

## Criterios de Aceptación

- The accepted format set is documented with examples.
- Projected query output has stable ordering across runs.
- CSV/TSV, if implemented, handle commas, tabs, newlines, and nil values deterministically.
- JSONL, if implemented, emits one valid JSON object per line.
- `go test ./cmd/rootline` passes.

## Fuente de verdad

- `cmd/rootline/root.go`
- `cmd/rootline/query.go`
- `cmd/rootline/table.go`
- `README.md`
- `docs/query.md`
