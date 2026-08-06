---
estado: Completed
---
# Statistics

`rootline stats` counts records in a directory tree. Use `--where` to filter which records are included in the total.

## CLI Usage

```bash
rootline stats                                    # Current directory
rootline stats docs/epics/                        # Specific path
rootline stats --from docs/epics/                 # Explicit root
rootline stats --where 'estado != "Completed"'    # Filtered
rootline stats -o json                            # JSON output (default)
rootline stats -o table                           # Summary line
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
  "version": 2,
  "kind": "rootline/stats",
  "total": 21
}
```

The JSON result contains only the versioned command kind and the filtered total. Use `--where` for field-specific slices.
`stats` supports `-o json` and `-o table` only. `-o jsonl` and `-o csv` are rejected — see [Output Formats](output.md).
