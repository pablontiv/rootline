---
estado: Completado
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
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Report changes without modifying files |
| `--from` | Compare against specified `.stem` file instead of git HEAD |
| `--rename old=new` | Rename a field across all documents and `.stem` files |

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
