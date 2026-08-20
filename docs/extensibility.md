---
estado: Completed
---
# Extensibility

Rootline is built around **extractors**.

Markdown is the built-in extractor.

The architecture is designed so that other extractors can be added
without changing the core model. Future extractors may include:

- YAML / JSON / TOML files
- MDX
- API specifications (OpenAPI, AsyncAPI)

All extractors feed the same pipeline:
rules by directory, inheritance, validation, derivation, querying.

## Schema Field Types

The `.stem` schema supports the following canonical field types:

| Type | Description |
|------|-------------|
| `string` | Free-text value |
| `list` | YAML sequence |
| `enum` | One value from `values:` |
| `sequence` | Identifier with a prefix and exact decimal digits |
| `link` | Wiki-link string or list of wiki-link strings |
| `boolean` | Native YAML boolean |
| `integer` | Native YAML integer |

Types are strict: Rootline does not coerce quoted booleans or numbers. The canonical spellings are `type: boolean` and `type: integer`; a one-value enum is valid, for example `values: [theory]`.

## Body Sources

A Markdown location is declared separately from its value type:

```yaml
schema:
  summary:
    type: string
    source: body.section["## Summary"]
    required: true
  changelog:
    type: string
    source: body.section["## Changelog"]
    required: false
    default: "<!-- TODO -->"
```

`source: body.h1` and exact `body.section[...]` directives are supported. Frontmatter is an explicit override. An empty section is present with value `""`; duplicate matching headings fail rather than selecting one. Extractors preserve source identity so validation, inference, schema proposal, and schema application use the same canonical directive.

Source-backed fields participate in validation, querying, describe/explain output, and scaffolding. `rootline set` writes a frontmatter override for such a field; it does not edit the body section.

> LSP integration has been considered but carries very high complexity.
> It is not in scope.
