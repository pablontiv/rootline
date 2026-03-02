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
rootline migrate --from-levels            # Convert v1 levels: to v2 match:-based fields
rootline migrate --from-levels --dry-run  # Preview conversion
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Report changes without modifying files |
| `--from` | Compare against specified `.stem` file instead of git HEAD |
| `--rename old=new` | Rename a field across all documents and `.stem` files |
| `--split` | Split a flat `.stem` into hierarchical `.stem` files per level |
| `--from-levels` | Convert v1 `.stem` with `levels:` to v2 with `match:`-based fields |

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

## From-Levels Migration

`--from-levels` converts a v1 `.stem` that uses the `levels:` keyword into a v2 `.stem` with `match:`-based field annotations:

```bash
rootline migrate --from-levels docs/epics/
```

The migration reads each level's schema fields and converts them to flat schema fields with `match:` annotations. Sequence fields get per-pattern configs in their `match:` map. The `levels:` section is removed and `version` is set to `2`.

See [Hierarchical Schema](levels.md) for the v2 format.
