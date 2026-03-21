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

The `.stem` schema supports the following field types:

| Type | Description |
|------|-------------|
| `string` | Free-text value |
| `enum` | One of a fixed set of values |
| `sequence` | Auto-incrementing identifier with prefix/digits |
| `section` | A Markdown H2/H3 heading and its body in the document |

### `type: section`

`type: section` makes a Markdown heading a first-class schema field. The Markdown extractor populates `Record.Sections` — a `map[string]string` from heading text to body — when AST extraction is active.

```yaml
schema:
  "## Summary":
    type: section
    required: true
  "## Changelog":
    type: section
    required: false
    default: "<!-- TODO -->"
    ordered: false
```

**Properties**:

| Property | Type | Description |
|----------|------|-------------|
| `heading` | string | The heading text to match (defaults to the field key) |
| `required` | bool | Whether the section must be present in every document |
| `default` | string | Default body content for `migrate --scaffold` |
| `ordered` | bool | If true, validate that the section appears in schema-defined order |

Section fields participate in validation (required check), querying (`sections["## Heading"]`), and mutation (`rootline set '## Heading=content'`). They are emitted by `rootline init` at the 0.80 presence threshold.

> LSP integration has been considered but carries very high complexity.
> It is not in scope.
