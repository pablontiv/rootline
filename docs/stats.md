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

`stats` is field-agnostic: it reports the total record count and makes no assumptions about which frontmatter fields exist:

```
Total: 21 records
```

For field-specific counts, filter with `--where` and compare totals:

```bash
rootline stats docs/roadmap/ --where "estado == 'Completed'"
rootline stats docs/roadmap/ --where "tipo == 'software'"
```

## JSON Result

```json
{
  "version": 1,
  "kind": "rootline/stats",
  "by_lifecycle_state": {},
  "by_record_type": {},
  "total": 21
}
```

The `by_lifecycle_state` and `by_record_type` maps are reserved in the output contract but are currently always empty; use `--where` for field-specific slices.
