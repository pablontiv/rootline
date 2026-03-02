---
estado: Completed
---
# Explain

`rootline explain` traces the origin of every field in a document: which `.stem` defined it, whether it came from frontmatter, derivation, or aggregation, and what expressions produced the value.

## CLI Usage

```bash
rootline explain docs/query.md
rootline explain docs/epics/E04/README.md -o json
```

## JSON Result

```json
{
  "version": 1,
  "kind": "rootline/explain",
  "path": "docs/query.md",
  "stem_chain": ["docs/.stem"],
  "fields": [
    {
      "name": "estado",
      "value": "Completed",
      "origin": "frontmatter",
      "source": "docs/.stem"
    }
  ]
}
```

For documents with derived and aggregated fields, each field shows its expression and origin:

```json
{
  "version": 1,
  "kind": "rootline/explain",
  "path": "docs/epics/E04/README.md",
  "stem_chain": ["docs/epics/.stem"],
  "fields": [
    {
      "name": "estado",
      "value": "In Progress",
      "origin": "frontmatter",
      "source": "docs/epics/.stem"
    },
    {
      "name": "slug",
      "value": "dx-advanced",
      "origin": "derived",
      "expression": "slugify(titulo)"
    },
    {
      "name": "total",
      "value": 15,
      "origin": "aggregate",
      "expression": "len(descendants)"
    }
  ],
  "errors": [
    {
      "rule": "enum",
      "field": "tipo",
      "message": "invalid value \"unknown\"",
      "source": "docs/epics/.stem",
      "severity": "error"
    }
  ]
}
```

The `stem_chain` shows the walk-up discovery order from target to `.git` root. The `origin` field is one of: `frontmatter`, `schema` (default value), `derived`, or `aggregate`.
