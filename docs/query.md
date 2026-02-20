---
estado: Completado
---
# Query Engine

Rootline exposes data through a **declarative query model**.

Queries return **records**, not rendered documents.

## Query Request Contract

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

When `where` contains multiple conditions without an explicit `and` wrapper,
they are combined with `and` semantics (implicit `and`).

## Operators

| Operator | Semantics | Example |
|----------|-----------|---------|
| `eq` | Equals | `{"eq": ["status", "published"]}` |
| `ne` | Not equals | `{"ne": ["status", "draft"]}` |
| `in` | One of | `{"in": ["tipo", ["lxc", "vm"]]}` |
| `contains` | Substring match | `{"contains": ["body", "migration"]}` |
| `exists` | Field is present | `{"exists": "owner"}` |
| `and` | All conditions match | `{"and": [cond1, cond2]}` |

Query functions: `limit`, `count`.

- Queries are **purely declarative**
- No embedded code or scripts
- Stable semantics for automation and AI
- Operator set derived from real consumer analysis

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
      }
    }
  ]
}
```

## Count Result Shape

```json
{
  "version": 1,
  "kind": "rootline/count",
  "meta": {},
  "count": 12
}
```
