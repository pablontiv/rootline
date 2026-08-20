---
estado: Completed
---
# Derivation Engine

Rootline evaluates expressions declared in `.stem` files to compute **derived fields** at query time, without modifying source files.

Two mechanisms:
- **Derive**: per-record expressions (e.g., `slug: "slugify(titulo)"`)
- **Aggregate**: bottom-up expressions for index files (e.g., `total: "len(descendants)"`)

Both use [expr-lang/expr](https://expr-lang.org/) — non-Turing complete, sandboxed, 70ns/op.

## .stem Configuration

```yaml
derive:
  slug: "slugify(titulo)"
  status_lower: "lower(estado)"

aggregate:
  total: "len(descendants)"
  completed: "len(filter(descendants, .estado == 'Completed'))"
```

`derive:` expressions run per-record. `aggregate:` expressions run on index files (README.md) with access to `descendants` (all non-index records) and `children` (sub-index records).

## Builtin Functions

| Function | Description | Example |
|----------|-------------|---------|
| `slugify(s)` | URL-friendly slug | `slugify("Mi Título")` → `"mi-titulo"` |
| `lower(s)` | Lowercase | `lower("ABC")` → `"abc"` |
| `upper(s)` | Uppercase | `upper("abc")` → `"ABC"` |
| `trim(s)` | Trim whitespace | `trim("  hi  ")` → `"hi"` |
| `strlen(s)` | String length | `strlen("abc")` → `3` |
| `concat(...)` | Concatenate | `concat("a", "-", "b")` → `"a-b"` |

Standard expr-lang builtins are also available: `len`, `filter`, `map`, `any`, `all`, `contains`, `startsWith`, `endsWith`, `split`, `replace`.

## Linked Field Injection

When documents use wiki-links (`[[blocks:T003-task]]`), the derivation engine resolves linked records and injects their field values into expressions:

```yaml
# .stem links section defines field mapping
links:
  blocks:
    fields:
      estado: estado  # inject linked record's "estado" into expression env
```

This enables derived state from dependencies — e.g., a task is blocked if any linked blocker has `estado != "Completed"`.

## Built-in Derived Fields

Rootline automatically computes structural metadata fields before user-defined derivation runs. These fields are always available in `--where` expressions without any `.stem` configuration.

| Field | Type | Description |
|-------|------|-------------|
| `isIndex` | `bool` | `true` if the record is a directory index file (default: `README.md`, configurable via `.stem` `structural.subdirs.require_index`) |

Example usage:

```bash
# List only content records (exclude index files)
rootline query docs/epics/ --where 'isIndex == false'

# Count index files
rootline query docs/epics/ --where 'isIndex == true' --count
```

## Pipeline

```
Extraction → Validation → Derivation → Enrichment → Aggregation → Query
```

Enrichment computes built-in fields (`isIndex`). Aggregation processes deepest indices first (bottom-up), so parent index files see fully-derived children.

## Tracing with Explain

`rootline explain` traces the origin of every field:

```bash
rootline explain docs/epics/E04-dx-advanced/README.md
```

Each field shows its origin: `frontmatter`, `schema` (default), `derived`, or `aggregate`. Physical schema provenance is reported as `defined_in`; the logical `source` field is used only for extraction directives such as `body.section[...]`. In plain-text explain output these correspond to `origin: derived` and `origin: aggregate` tokens.

## In Query Results

Derived and aggregated fields appear in query results alongside frontmatter:

```json
{
  "path": "docs/api/endpoints.md",
  "type": "markdown",
  "frontmatter": { "titulo": "Endpoints", "estado": "Completed" },
  "derived": { "slug": "endpoints", "status_lower": "completed" }
}
```
