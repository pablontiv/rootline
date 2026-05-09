---
estado: Completed
---
# Query Engine

Rootline exposes data through a **declarative query model**.

Queries return **records**, not rendered documents. Derived and aggregated fields are computed automatically and appear alongside frontmatter in results.

## CLI Usage

The `--where` flag accepts [expr-lang/expr](https://expr-lang.org/) expressions. Multiple `--where` flags are combined with AND:

```bash
rootline query --where 'status == "published"'
rootline query --where 'tipo in ["lxc", "vm"]' --where 'estado not in ["Completed"]'
```

### Sorting

The `--sort` flag accepts a comma-separated list of `field:direction` pairs. Direction defaults to `asc` if omitted:

```bash
rootline query --sort "prioridad:asc,impact_score:desc"
rootline query --where 'estado == "Pending"' --sort "prioridad:asc"
```

Sort type detection per field:
1. **Enum fields**: If the field has a `.stem` schema with `type: enum` and `values: [...]`, sorts by index position in the values list.
2. **Numeric fields**: If both values parse as float64, sorts numerically.
3. **String fields**: Fallback to lexicographic comparison.

Missing/nil values always sort last, regardless of direction. Sort applies after filtering and before limit.

### Compact Projections with `--select`

The `--select` flag produces compact output containing only specified fields:

```bash
rootline query --select path,estado,title
rootline query --select path,estado,title,links --output jsonl
rootline query --where 'estado == "Pending"' --select path,title --output csv
```

`--select` accepts a comma-separated list of field names. Supported field names include:
- `path` — document file path
- `title` — first Markdown heading or first non-empty body line
- Any frontmatter field name (e.g., `estado`, `prioridad`, `owner`)
- Any derived field (defined in `.stem` `derive:` section)
- `links` — document links extracted from wiki-link references

**Without `--select`**, query results include full records with `frontmatter`, `body`, `links`, and `derived`.

**With `--select`**, only specified fields are included in each row, reducing noise and making output suitable for shell pipelines.

### Output Formats

The `--output` flag supports multiple formats for projected query results:

| Format | Encoding | Use case |
|--------|----------|----------|
| `json` (default) | Full JSON objects in a `rows` array | AI agents, typed languages, complete data |
| `table` | Human-readable columnar format | Terminal inspection, reports |
| `jsonl` | One JSON object per line | Streaming processors, log ingestion |
| `csv` | RFC 4180 CSV with headers | Spreadsheets, SQL import, `awk`/`cut` pipelines |

> **Note**: `jsonl` and `csv` formats require `--select` to specify which columns to output.

Examples:

```bash
# JSON (default): full records in a rows array
rootline query --select path,estado --output json
# {"version": 1, "kind": "rootline/query", "rows": [{"path": "...", "estado": "..."}, ...]}

# JSONL: one record per line, no outer wrapper
rootline query --select path,estado --output jsonl
# {"path": "...", "estado": "Pending"}
# {"path": "...", "estado": "In Progress"}

# CSV: RFC 4180 with quoted fields and escaping
rootline query --select path,estado,title --output csv
# path,estado,title
# "docs/api/auth.md","Pending","Authentication Guide"
# "docs/api/endpoints.md","In Progress","Endpoint Reference"

# CSV with nil values (rendered as empty)
rootline query --select path,estado,owner --output csv
# path,estado,owner
# "docs/api/auth.md","Pending","alice"
# "docs/api/endpoints.md","In Progress",""
```

Column order in `jsonl` and `csv` output follows the order specified in `--select`.

### Queryable Fields

| Field | Type | Description |
|-------|------|-------------|
| `frontmatter.*` | any | Any frontmatter key |
| `body` | string | Full document body text |
| `sections` | map[string]string | Map of heading → section body (requires AST extraction) |
| `derived.*` | any | Computed derived fields from `.stem` `derive:` |

### Section Queries

When AST extraction is enabled, `sections` exposes a map of heading text to section body content:

```bash
rootline query --where 'sections["## Summary"] contains "migration"'
rootline query --where 'sections["## Changelog"] != nil'
rootline query --where 'sections["## References"] contains "[[T001]]"'
```

The heading key must match the heading text exactly, including the `#` prefix and any leading/trailing spaces.

> **Universal Filtering**: The `--where` flag is not limited to `query`. It is also available on **`tree`**, **`stats`**, **`graph`**, and **`validate --all`**. All transversal commands share the same expr-lang syntax.

> **Field Warnings**: Unknown field names in `--where` expressions emit warnings to stderr with fuzzy suggestions (e.g., `warning: unknown field "estdo" in where expression (did you mean "estado"?)`). Queries still execute — warnings are informational only.

## Operators

Standard expr-lang operators apply:

| Operator | Semantics | Example |
|----------|-----------|---------|
| `==` | Equals | `status == "published"` |
| `!=` | Not equals | `status != "draft"` |
| `in` | One of | `tipo in ["lxc", "vm"]` |
| `not in` | Not one of | `estado not in ["draft", "obsoleto"]` |
| `contains` | Substring match | `body contains "migration"` |
| `!= nil` | Field exists | `tags != nil` |
| `&&` | AND | `tipo == "lxc" && estado == "Pending"` |
| `\|\|` | OR | `estado == "Pending" \|\| estado == "Bloqueada"` |

## Query Request Contract (JSON)

```json
{
  "version": 1,
  "from": "docs/",
  "where": {
    "and": [
      {"eq": ["frontmatter.status", "published"]},
      {"exists": "frontmatter.owner"}
    ]
  },
  "limit": 50
}
```

The JSON contract uses declarative operators (`eq`, `ne`, `in`, `contains`, `exists`, `and`) for programmatic use. The CLI `--where` flag uses expr-lang syntax instead.

## Query Result Shape

```json
{
  "version": 1,
  "kind": "rootline/query",
  "meta": {
    "count": 1
  },
  "rows": [
    {
      "path": "docs/api/endpoints.md",
      "type": "markdown",
      "frontmatter": {
        "title": "Endpoints",
        "status": "published"
      },
      "derived": {
        "slug": "endpoints"
      }
    }
  ]
}
```

Rows include `derived` when the effective `.stem` defines `derive:` or `aggregate:` fields.

## Count Result Shape

```json
{
  "version": 1,
  "kind": "rootline/count",
  "meta": {},
  "count": 12
}
```

## AI Agent Compatibility

Some AI coding agents (notably Claude Code's Bash tool) pre-escape the `!` character to `\!` in command strings before shell processing. This breaks `!=` expressions at the expr-lang parser level with error `unrecognized character U+005C '\'`.

**Affected operators**: `!=`, `!= nil`, and any expression containing `!`.

**Workarounds** (in order of preference):

1. **`not in` operator** — Direct replacement for `!=` negation:
   ```bash
   # Instead of:
   rootline query --where 'estado != "draft"'
   # Use:
   rootline query --where 'estado not in ["draft"]'
   ```

2. **`in` with positive list** — Enumerate accepted values:
   ```bash
   rootline query --where 'estado in ["pendiente", "parcial"]'
   ```

3. **JSON + jq pipeline** — Query all, filter externally:
   ```bash
   rootline query -o json | jq '.rows[] | select(.estado != "draft")'
   ```

**Field existence check** (`!= nil`): There is no `not in` equivalent for field existence. Use the JSON pipeline approach:

```bash
rootline query -o json | jq '.rows[] | select(.tags != null)'
```

This issue is not specific to rootline — it affects any CLI tool whose query syntax uses `!` when invoked through agents with this escaping behavior.
