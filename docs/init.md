---
estado: Completed
---
# Schema Inference

`rootline init` scans existing Markdown files and infers a `.stem` schema from frontmatter patterns and supported Markdown body sources. It detects field types, enum values, required fields, and frequent section sources automatically.

## CLI Usage

```bash
rootline init docs/api/                           # Infer and write .stem
rootline init --dry-run docs/                     # Preview without writing
rootline init --force docs/api/                   # Overwrite existing .stem
```

Writes are atomic per file: Rootline stages each generated or template `.stem`
beside its destination and replaces the destination only after the complete
content is flushed. A multi-file template or hierarchical initialization is not
a transaction; files completed before a later error remain installed.

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Print inferred `.stem` to stdout without writing |
| `--force` | Overwrite existing `.stem` file |

## Flat Mode

When no hierarchical naming pattern is detected, init produces a simple schema:

```yaml
version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Completed, In Progress, Pending, Pre-research]
  fecha:
    type: string
  metodo:
    type: string
```

Fields present in all files are marked `required`. Fields with a small set of distinct values are inferred as `enum` using `values:`.

## Hierarchical Mode

When directory names follow patterns like `E##-*`, `F##-*`, `S###-*`, `T###-*`, init generates a `version: 2` schema with `match:`-based per-level fields and sequence auto-numbering:

```yaml
version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Completed, Pending, Specified]
  id:
    type: sequence
    match:
      "E*": { prefix: E, digits: 2 }
      "F*": { prefix: F, digits: 2 }
      "S*": { prefix: S, digits: 3 }
      "T*": { prefix: T, digits: 3 }
  ejecutable_en:
    type: string
    required:
      match: "T*"
    match: "T*"
aggregate:
  estado: |
    ...
```

Fields present at all levels stay global; fields present at specific levels get `match:` annotations. Aggregate expressions are auto-generated for enum fields on index files.

## Section Source Inference

When AST extraction is enabled, `rootline init` scans document bodies and infers source-backed fields for Markdown headings that appear frequently across files.

Init uses a **0.80 threshold** — a heading must appear in at least 80% of files to be emitted. This is stricter than `analyze` default threshold of 0.60, to avoid generating spurious required sections from document-specific headings.

### Example

Given a directory where every file has a `## Summary` heading and 85% have `## Changelog`:

```yaml
version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Completed, In Progress, Pending]
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

Headings below the 0.80 threshold are omitted from the generated `.stem`. The source preserves exact heading level and text. Frontmatter remains an explicit override, and generated schemas validate the source corpus; distinct exact headings that normalize to one logical field fail with a collision instead of receiving an invented suffix.

### Source-Backed Field Properties

| Property | Description |
|----------|-------------|
| `type` | Real value type, usually `string` for Markdown section content |
| `source` | Exact directive such as `body.section["## Summary"]` |
| `required` | Whether the source-backed field must be present in every applicable document |
| `default` | Content inserted when `rootline migrate --scaffold` or `rootline new` adds the section |
