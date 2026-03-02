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

The title is auto-generated from the filename: dashes and underscores become spaces, with title case applied. Required fields appear with empty placeholders. Enum fields include the allowed values as inline comments when defaults are available.

## Sequence Fields

When the target directory has a `sequence` field in the schema, `rootline new` auto-generates the next available number based on existing files in the directory:

```bash
rootline new docs/epics/E04/F03/S001/T004-new-task.md
# id field auto-set to T004
```
