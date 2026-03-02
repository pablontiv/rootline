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
