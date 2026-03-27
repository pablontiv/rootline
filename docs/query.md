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
rootline query --where 'tipo in ["lxc", "vm"]' --where 'estado != "Completed"'
```

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
