---
estado: Completed
---
# Schema Migration

When `.stem` files change, existing documents may become invalid. `rootline migrate` detects breaking changes and performs bulk migration operations.

Complementary to `rootline fix` — fix corrects data errors, migrate handles schema evolution.

## CLI Usage

```bash
rootline migrate                          # Diff current .stem vs git HEAD
rootline migrate --dry-run                # Report changes without modifying files
rootline migrate --from old.stem          # Compare against specific .stem file
rootline migrate --rename old_field=new   # Rename field across all documents + .stem files
rootline migrate --split                  # Split flat .stem into hierarchical per-level files
rootline migrate --split --dry-run        # Preview split without writing files
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Report changes without modifying files |
| `--from` | Compare against specified `.stem` file instead of git HEAD |
| `--rename old=new` | Rename a field across all documents and `.stem` files |
| `--split` | Split a flat `.stem` into hierarchical `.stem` files per level |
| `--scaffold` | Scaffold missing required sections into documents using schema defaults |

## Change Detection

Diff mode compares `.stem` files and classifies each change:

| Change Kind | Breaking? | Example |
|-------------|-----------|---------|
| `field_added` | No | New optional field in schema |
| `field_removed` | Yes | Field deleted from schema |
| `type_changed` | Yes | Field type changed (e.g., string → enum) |
| `enum_value_added` | No | New allowed value |
| `enum_value_removed` | Yes | Value no longer valid |
| `required_added` | Yes | Field now required |
| `required_removed` | No | Field no longer required |
| `default_changed` | No | Default value changed |
| `severity_changed` | Depends | Severity tightened or loosened |
| `rule_added` / `rule_removed` | Depends | Validation rule added or removed |

## Result Shape

```json
{
  "version": 1,
  "kind": "rootline/migrate",
  "changes": [
    {
      "kind": "field_removed",
      "field": "prioridad",
      "breaking": true,
      "before": "enum",
      "after": null,
      "message": "field 'prioridad' removed",
      "affected_files": 12
    }
  ],
  "breaking_count": 1,
  "total_count": 1
}
```

## Rename Mode

`--rename` updates a field name across all documents and `.stem` files atomically:

```bash
rootline migrate --rename status=estado
```

Updates frontmatter in all affected markdown files and schema definitions in `.stem` files. Operations are logged to `.migration-log.json` (JSON Lines, append-only).

## Split Mode

`--split` converts a flat `.stem` into hierarchical per-level `.stem` files. It detects directory naming patterns (e.g., `E##-*`, `F##-*`, `S###-*`, `T###-*`) and distributes schema fields by real presence at each level.

```bash
rootline migrate --split docs/epics/       # Split into per-level .stem files
rootline migrate --split --dry-run docs/   # Preview without writing
```

### How it works

1. Scans records and detects hierarchy levels from directory naming patterns
2. Analyzes which fields have values at which levels
3. Fields present at **all levels** stay in the root `.stem`
4. Fields present at **some levels** go to per-level `.stem` files
5. Each level gets a `sequence` id field matching its prefix/digits
6. `derive`, `aggregate`, `links`, `structural`, and `validate` rules are preserved at root

### Example

Given a flat `.stem` with fields used across `E##/F##/S###/T###` directories:

```
docs/epics/.stem              → root fields + E-level sequence + derive/aggregate/links
docs/epics/E03-name/.stem     → F-level sequence + F-only fields
docs/epics/E03-name/F01-x/.stem → S-level sequence + S-only fields
```

Requires at least 2 hierarchy levels to be detected. Use `--dry-run` to preview the split before applying.

## Scaffold Mode

`--scaffold` adds missing required sections to documents that do not have them. It uses the `default` property of each `type: section` field in the effective `.stem` to populate inserted content.

```bash
rootline migrate --scaffold docs/epics/          # Add missing required sections
rootline migrate --scaffold --dry-run docs/      # Preview without writing
```

### Content Priority

When inserting a missing section, content is selected in this order:

1. `default` value from the `type: section` field in the effective `.stem`
2. `"<!-- TODO -->"` (fallback when no default is defined)

### Example

Given a `.stem` with:

```yaml
schema:
  "## Summary":
    type: section
    required: true
    default: "Describe the purpose of this document."
  "## References":
    type: section
    required: true
```

Running `rootline migrate --scaffold` on a file missing both sections produces:

```
~ docs/epics/E03/README.md
  + ## Summary
    Describe the purpose of this document.
  + ## References
    <!-- TODO -->
```

Sections are inserted at the end of the document body. Use `--dry-run` to review insertions before applying.
