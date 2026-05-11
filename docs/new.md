---
estado: Completed
---
# Document Scaffolding

`rootline new` creates a markdown file with frontmatter pre-populated from the effective `.stem` schema of the target directory.

## CLI Usage

```bash
rootline new docs/api/new-endpoint.md             # Create file
rootline new --dry-run docs/api/new-endpoint.md   # Preview content
rootline new --force docs/api/existing.md         # Overwrite existing
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Show generated content without writing file |
| `--force` | Overwrite existing file |

## Output Example

```markdown
---
estado:
---
# New Endpoint
```

The title is auto-generated from the filename: dashes and underscores become spaces, with title case applied.

## Enum Fields

Enum fields are always scaffolded with an inline comment listing the allowed values:

- Fields with an explicit `default:` in the schema use that default as the initial value.
- Fields without an explicit default are scaffolded with an empty value so the user must choose:

```markdown
tipo:  # [outcome, task]
```

This prevents wrong defaults from being written silently.

## Sequence Fields

Sequence fields (`type: sequence`) are scaffolded with an empty placeholder. Auto-generation of the next sequence value is not supported by `rootline new`; use `rootline describe <dir>` to find the next available ID:

```bash
rootline describe docs/epics/ --field schema.id.next
```
