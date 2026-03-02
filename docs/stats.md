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
  "by_estado": {
    "Completed": 10,
    "In Progress": 1,
    "Pending": 1,
    "Pre-research": 4
  },
  "by_tipo": {
    "software": 12,
    "infra": 5,
    "docs": 4
  },
  "total": 21
}
```
