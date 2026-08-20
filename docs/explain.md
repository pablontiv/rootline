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
      "defined_in": "docs/.stem"
    },
    {
      "name": "notes",
      "value": "Release notes go here.",
      "origin": "derived",
      "source": "body.section[\"## Notes\"]",
      "defined_in": "docs/.stem"
    },
    {
      "name": "total",
      "value": 3,
      "origin": "aggregate",
      "expression": "len(descendants)",
      "defined_in": "docs/.stem"
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
      "defined_in": "docs/epics/.stem"
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
      "severity": "error",
      "suggestion": "task"
    }
  ]
}
```

The `stem_chain` shows the walk-up discovery order from the target up to the `root: true` marker. A chain that reaches the filesystem root without one has no declared boundary; the preflight stops governed commands there and asks for the marker. The `origin` field is one of: `frontmatter`, `schema` (default value), `derived`, or `aggregate`.

For a source-backed field, `source` is the logical directive and `defined_in` is the physical `.stem` declaration. Explain resolves the same effective value as validation and query; frontmatter remains an override, empty-present sections remain present, and duplicate matching headings are reported as errors.

`explain` supports `-o json` and `-o table` only — see [Output Formats](output.md).
