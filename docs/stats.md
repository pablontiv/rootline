---
estado: Completed
---
# Statistics

`rootline stats` shows aggregate counts by `estado` and `tipo` frontmatter fields.

## CLI Usage

```bash
rootline stats                                    # Current directory
rootline stats docs/epics/                        # Specific path
rootline stats --from docs/epics/                 # Explicit root
rootline stats --where 'estado != "Completed"'    # Filtered
rootline stats -o json                            # JSON output
```

### Flags

| Flag | Description |
|------|-------------|
| `--from` | Root path to scan (default: `.`) |
| `--where "expr"` | Filter records (expr-lang, repeatable) |

## Table Output

When documents define `estado` and `tipo` fields in `.stem`, the table shows aggregates by those fields:

```
Total: 21 records

By Estado:
  Completed            10
  In Progress          1
  Pending              1
  Pre-research         4

By Tipo:
  software             12
  infra                5
  docs                 4
```

## JSON Result

```json
{
  "version": 1,
  "kind": "rootline/stats",
  "by_lifecycle_state": {
    "Completed": 10,
    "In Progress": 1,
    "Pending": 1,
    "Pre-research": 4
  },
  "by_record_type": {
    "software": 12,
    "infra": 5,
    "docs": 4
  },
  "total": 21
}
```

When no consistent `estado` or `tipo` fields are found across documents, the maps are empty.
