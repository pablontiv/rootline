---
estado: Completado
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
  completados: "len(filter(descendants, .estado == 'Completado'))"
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

This enables derived state from dependencies — e.g., a task is blocked if any linked blocker has `estado != "Completado"`.

## Pipeline

```
Extraction → Validation → Derivation → Aggregation → Query
```

Aggregation processes deepest indices first (bottom-up), so parent index files see fully-derived children.

## Tracing with Explain

`rootline explain` traces the origin of every field:

```bash
rootline explain docs/epics/E04-dx-advanced/README.md
```

Each field shows its origin: `frontmatter`, `schema` (default), `derived`, or `aggregated`, with the source `.stem` file.

## In Query Results

Derived and aggregated fields appear in query results alongside frontmatter:

```json
{
  "path": "docs/api/endpoints.md",
  "type": "markdown",
  "frontmatter": { "titulo": "Endpoints", "estado": "Completado" },
  "derived": { "slug": "endpoints", "status_lower": "completado" }
}
```
