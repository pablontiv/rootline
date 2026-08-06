---
estado: Completed
---
# Schema Inference

`rootline init` scans existing markdown files and infers a `.stem` schema from frontmatter patterns. It detects field types, enum values, and required fields automatically.

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

Fields present in all files are marked `required`. Fields with a small set of distinct values are inferred as `enum`.

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

## Section Field Inference

When AST extraction is enabled, `rootline init` also scans document bodies and infers `type: section` fields for Markdown headings that appear frequently across files.

Init uses a **0.80 threshold** — a heading must appear in at least 80% of files to be emitted as a section field. This is stricter than `analyze`'s default threshold of 0.60, to avoid generating spurious required sections from document-specific headings.

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
  "## Summary":
    type: section
    required: true
  "## Changelog":
    type: section
    required: false
    default: "<!-- TODO -->"
```

Headings below the 0.80 threshold are omitted from the generated `.stem`.

### Section Field Properties

| Property | Description |
|----------|-------------|
| `type: section` | Marks a Markdown H2/H3 heading as a first-class schema field |
| `heading` | The heading string (defaults to the field key) |
| `required` | Whether the section must be present in every document |
| `default` | Content inserted when `rootline migrate --scaffold` adds the section |
| `ordered` | If true, validate that the section appears in schema-defined order |
