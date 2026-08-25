# Query, Exploration, and Explain Reference

## Shared Filter Syntax

Commands with `--where` use expr syntax:

```bash
rootline query <dir> --where "estado == 'Pending'"
rootline query <dir> --where "estado in ['Pending', 'Specified']"
rootline query <dir> --where "tipo == 'task' && estado != 'Completed'"
rootline query <dir> --where "body contains 'migration'"
rootline query <dir> --where "tags != nil"
```

Supported operators: `==`, `!=`, `in`, `not in`, `contains`, `&&`, `||`. Use `field != nil` or `field not in [nil]` for existence checks. Built-in fields include `path`, `body`, `type`, `sections`, and `isIndex` where available. The `type` field refers to the record type (e.g., `"markdown"`); it is not a function call.

## query

Use `query` to return records matching frontmatter, derived, aggregated, body, or built-in fields.

### Usage

```bash
rootline query <dir> -o json
rootline query <dir> --where "estado == 'Pending'" -o json
rootline query <dir> --where "isIndex == false" --limit 10 -o json
rootline query <dir> --count -o json
rootline query <dir> --sort "prioridad:asc,impact_score:desc" -o json
```

`query [path]` and `--from <path>` both set the scan root. Prefer the positional path.

### Flags

| Flag | Use |
|---|---|
| `--where "expr"` | filter records; repeated flags are ANDed |
| `--count` | return only count metadata |
| `--limit N` | limit rows after filtering and sorting |
| `--sort "field:asc,other:desc"` | deterministic multi-key sort |
| `--from <path>` | scan root when no positional path is used |
| `--select "path,estado,titulo"` | compact projection; include only named fields |
| `--has-inbound "expr"` | keep records with an inbound link from a record matching expr (`""` = any) |
| `--has-outbound "expr"` | keep records with an outbound link to a record matching expr (`""` = any) |
| `--inbound-type <t>` / `--outbound-type <t>` | restrict traversal edges to one link type |
| `--graph-root <path>` | edge-scan universe for traversal (default: the query path) |

### Output Formats

Default (`--output json`) returns structured JSON. With `--select`, use `--output jsonl` or `--output csv` for streaming or processing convenience. `query` is the only command that implements all four formats; elsewhere `jsonl`/`csv` are rejected rather than downgraded to JSON.

**Field-name validation**: an unknown field in `--where` prints `warning: unknown field "estdo" in where expression (did you mean "estado"?)` on stderr and leaves the exit code alone — on `query`, `stats`, `tree`, `graph` and `validate --all`. An unknown field in `--sort` is an **error** (rc=1), like a bad sort direction. Legal names are the query builtins, every key the corpus carries, and every `schema:`/`derive:`/`aggregate:` name the `.stem` chain declares. Treat a zero-result query with no warning as a genuine empty set.

#### JSON (default)

Without `--select` (full records):
```json
{
  "version": 1,
  "kind": "rootline/query",
  "rows": [
    {
      "path": "docs/T001-task.md",
      "frontmatter": { "estado": "Pending" },
      "derived": {}
    }
  ],
  "meta": { "count": 1 }
}
```

With `--select "path,estado,titulo"` (compact projection):
```json
{
  "version": 1,
  "kind": "rootline/query",
  "rows": [
    {
      "path": "docs/T001-task.md",
      "estado": "Pending",
      "titulo": "T001: some task"
    }
  ],
  "meta": { "count": 1 }
}
```

#### JSONL (with `--select` only)

Emit one JSON object per line — useful for shell pipes and streaming:

```bash
rootline query docs/roadmap --select path,estado --output jsonl
```

Output:
```
{"estado":"Pending","path":"docs/T001-task.md"}
{"estado":"Completed","path":"docs/T002-task.md"}
```

Pipe to `jq` for further processing:
```bash
rootline query docs/roadmap --select path,estado --output jsonl | jq -r '.path'
```

#### CSV (with `--select` only)

Emit CSV with header row — columns follow `--select` field order:

```bash
rootline query docs/roadmap --select path,estado,titulo --output csv
```

Output:
```
path,estado,titulo
docs/T001-task.md,Pending,T001: some task
docs/T002-task.md,Completed,T002: another task
```

Missing or nil fields render as empty columns. CSV quoting handles commas, tabs, and newlines automatically.

### Projection with --select

Use `--select` for compact output with only specified fields. Examples:

```bash
rootline query docs/roadmap --select path,estado,titulo
rootline query docs/roadmap --where "estado == 'In Progress'" --select path,titulo,links
```

Fields available in projection:
- `path` — document path relative to scan root
- `estado`, `tipo`, etc. — any frontmatter field
- `titulo` — derived field from `.stem` source extraction (e.g., `source: body.h1`)
- `links` — array of links extracted from document body: both `[[wiki-links]]` and `[markdown](links)` references (which styles are active depends on `.stem links.styles`)
- Missing fields are omitted from the projected row

Derived fields are populated via `.stem` `source:` rules. For example, `titulo: {source: body.h1}` extracts the first Markdown heading. If a source cannot be extracted, the field is omitted from the row.

### Source-Backed Fields

`query` resolves `source: body.section["## Heading"]` through the effective schema. The field has a real type; frontmatter is an explicit frontmatter override. An empty section is present, while duplicate matching headings fail instead of returning one value. A child may omit and inherit a source binding, but cannot change it.

## Link-traversal predicates

Use `--has-inbound` / `--has-outbound` to filter records by their RELATIONS: a record is kept when at least one linked record matches the sub-expression (same syntax as `--where`, evaluated against the linked record — the source for inbound, the target for outbound). An empty expression (`""`) means "any linked record" (existence check). Predicates AND-compose with `--where` and apply before `--sort`/`--limit`/`--count`.

```bash
# entities of kind tool with a corroborated witness linking to them via a `supports` edge
rootline query wiki/entities \
  --where "kind == 'tool'" \
  --has-inbound "verification == 'corroborated'" \
  --inbound-type supports \
  --graph-root wiki -o json

# sources that link to at least one tool
rootline query wiki/sources --has-outbound "kind == 'tool'" --graph-root wiki -o json

# records with any inbound link at all
rootline query wiki/entities --has-inbound "" --graph-root wiki -o json
```

Rules:

- `--graph-root` sets the universe for the edge scan (inbound links usually live OUTSIDE the queried directory). It defaults to the query path — never the repo root — and the query path must lie inside it. Choose it to exclude archived or raw trees.
- With traversal active, record paths in output stay relative to the query path (same format as a non-traversal query), and links are prepared exactly like `graph` (styles filtering + markdown target resolution). Broken links never satisfy a predicate.
- `--inbound-type`/`--outbound-type` require their `--has-*` flag; `--graph-root` requires at least one predicate.
- Output keeps the standard `rootline/query` version 1 envelope and composes with `--select`, `--output`, `--sort`, `--limit`, `--count`.

## tree

Use `tree` to show hierarchy and recursive record totals.

```bash
rootline tree <dir> -o table
rootline tree <dir> --where "estado != 'Completed'" -o table
rootline tree <dir> --where "isIndex == false" -o json
```

Use table output for humans and JSON for programmatic handling. Table format (`-o table`) renders ASCII brackets with the lifecycle field value (e.g., `[Completed]`, `[In Progress]`). The field name is resolved from the stem schema — not hardcoded to `"estado"`. JSON output (`-o json`) is the default and returns version 2. Leaf document nodes carry `frontmatter` with derived/aggregate values merged into it; directory nodes carry `children` and recursive `total`, not `frontmatter`.

## stats

Use `stats` for aggregate counts.

```bash
rootline stats <dir> -o json
rootline stats <dir> --where "tipo == 'task'" -o json
```

JSON contains counts grouped by `estado`, `tipo`, and total records.

## explain

Use `explain` when the user asks why a field has a value, where a schema-backed field is defined, or how derived/aggregate values are computed.

```bash
rootline explain <file.md> -o json
```

Report:

- `stem_chain` (root-to-leaf order)
- each field name/value
- origin: frontmatter, derived, aggregate, schema
- logical `source` and defined_in `.stem` provenance when present on schema-backed fields
- expression when present on derive/aggregate fields; current output does not include `defined_in` for those computed fields
- errors

