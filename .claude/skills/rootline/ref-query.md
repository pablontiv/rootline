# Query, Exploration, Explain, and Trace Reference

## Shared Filter Syntax

Commands with `--where` use expr syntax:

```bash
rootline query <dir> --where "estado == 'Pending'"
rootline query <dir> --where "estado in ['Pending', 'Specified']"
rootline query <dir> --where "tipo == 'task' && estado != 'Completed'"
rootline query <dir> --where "body contains 'migration'"
rootline query <dir> --where "tags != nil"
```

Supported operators: `==`, `!=`, `in`, `contains`, `&&`, `||`. Use `field != nil` for existence checks. Built-in fields include `path`, `body`, `sections`, and `isIndex` where available.

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

### Output Formats

Default (`--output json`) returns structured JSON. With `--select`, use `--output jsonl` or `--output csv` for streaming or processing convenience.

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
      "derived": {},
      "aggregated": {}
    }
  ],
  "meta": { "count": 1 }
}
```

With `--select "path,estado,title"` (compact projection):
```json
{
  "version": 1,
  "kind": "rootline/query",
  "rows": [
    {
      "path": "docs/T001-task.md",
      "estado": "Pending",
      "title": "T001: some task"
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
rootline query docs/roadmap --select path,estado,title --output csv
```

Output:
```
path,estado,title
docs/T001-task.md,Pending,T001: some task
docs/T002-task.md,Completed,T002: another task
```

Missing or nil fields render as empty columns. CSV quoting handles commas, tabs, and newlines automatically.

### Projection with --select

Use `--select` for compact output with only specified fields. Examples:

```bash
rootline query docs/roadmap --select path,estado,title
rootline query docs/roadmap --where "estado == 'In Progress'" --select path,title,links
```

Fields available in projection:
- `path` — document path relative to scan root
- `estado`, `tipo`, etc. — any frontmatter field
- `derived_field` — any derived field from `.stem`
- `title` — extracted from first Markdown heading (`# Heading`)
- `links` — array of wiki-link references
- Missing fields are omitted from the projected row

`title` is a special computed field extracted from the first `# Heading` in the document body. If no heading exists, it is omitted from the row.

## tree

Use `tree` to show hierarchy and completion counts.

```bash
rootline tree <dir> -o table
rootline tree <dir> --where "estado != 'Completed'" -o table
rootline tree <dir> --where "isIndex == false" -o json
```

Use table output for humans and JSON for programmatic handling.

## stats

Use `stats` for aggregate counts.

```bash
rootline stats <dir> -o json
rootline stats <dir> --where "tipo == 'task'" -o json
```

JSON contains counts grouped by `estado`, `tipo`, and total records.

## explain

Use `explain` when the user asks why a field has a value, where a field is defined, or how derived/aggregated values are computed.

```bash
rootline explain <file.md> -o json
```

Report:

- `stem_chain`
- each field name/value
- origin: frontmatter, derived, aggregated, schema
- source `.stem`
- expression when present
- errors

## trace

Use `trace` for reference-chain traversal from one document. Use `graph` for whole-repo topology.

### Usage

```bash
rootline trace <file.md> --format json
rootline trace <file.md> --reverse --format json
rootline trace <file.md> --depth 2 --type blocks --format json
rootline trace <file.md> --format tree
```

`trace` requires a `.git` directory above the file because it computes paths relative to the repo root.

### Flags

| Flag | Use |
|---|---|
| `--reverse` | find documents that reference the starting file |
| `--depth N` | maximum BFS depth; `0` means unlimited |
| `--type <edge>` | follow only one wiki-link edge type |
| `--format tree|json` | output format |
