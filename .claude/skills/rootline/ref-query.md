# Queries & Exploration Reference

## query — Search and filter records

### Usage

```bash
rootline query                                            # All records
rootline query --where "estado == 'Pending'"              # Filter
rootline query --where "tipo == 'task'" --where "estado == 'Pending'"  # AND
rootline query --count                                    # Count only
rootline query --limit 10                                 # Limit results
rootline query --from docs/epics/                         # Scope to subtree
```

### Flags

| Flag | Description |
|------|-------------|
| `--where "expr"` | Filter expression (repeatable, AND'd together) |
| `--count` | Return count instead of records |
| `--limit N` | Limit number of results (0 = unlimited) |
| `--from <path>` | Root path to scan (default: `.`) |

### Expression syntax

Uses `expr-lang/expr`:
- Comparison: `==`, `!=`
- Membership: `in`, `contains`
- Existence: `exists`
- Boolean: `&&` (AND)
- Field access: direct field names from frontmatter and derived fields

### Built-in fields

Available in all `--where` expressions without `.stem` configuration:

| Field | Type | Description |
|-------|------|-------------|
| `isIndex` | `bool` | `true` for index files (README.md), `false` for content files |

```bash
rootline query --where 'isIndex == false'   # content records only
rootline tree --where 'isIndex == false'     # tree without index files
```

### Output format (`kind: "rootline/query"`):

```json
{
  "version": 1,
  "kind": "rootline/query",
  "rows": [
    {
      "path": "docs/epics/E01/features/F01.md",
      "frontmatter": { "estado": "Pending", "tipo": "feature" },
      "derived": { "slug": "f01-auth" }
    }
  ],
  "total": 5
}
```

Table output: dynamic columns based on all fields in the result set.

---

## tree — Hierarchical view with completion counts

### Usage

```bash
rootline tree                                    # From current directory
rootline tree <path>                             # From specified path
rootline tree --where "estado != 'Completed'"    # Filtered
```

### Flags

| Flag | Description |
|------|-------------|
| `--where "expr"` | Filter expression (repeatable) |

### Output format (`kind: "rootline/tree"`):

JSON: hierarchical node structure with completion counts per directory.

ASCII: Tree diagram with `[completed/total]` at each directory level and `[estado]` for leaf nodes.

```
docs/epics/ [3/10]
├── E01-auth/ [2/4]
│   ├── features/
│   │   ├── F01-login.md [Completed]
│   │   └── F02-signup.md [Pending]
```

---

## stats — Summary counts by type and state

### Usage

```bash
rootline stats                                   # All records
rootline stats --from docs/epics/                # Scoped
rootline stats --where "tipo == 'task'"          # Filtered
```

### Flags

| Flag | Description |
|------|-------------|
| `--from <path>` | Root path to scan (default: `.`) |
| `--where "expr"` | Filter expression (repeatable) |

### Output format (`kind: "rootline/stats"`):

```json
{
  "version": 1,
  "kind": "rootline/stats",
  "by_estado": { "Pending": 5, "Completed": 3 },
  "by_tipo": { "task": 4, "feature": 2, "historia": 2 },
  "total": 8
}
```

Table: Summary with "Total: N records", then "By Estado:" and "By Tipo:" sections.

---

## explain — Trace field derivation

### Usage

```bash
rootline explain <file>
```

### Output format (`kind: "rootline/explain"`):

```json
{
  "version": 1,
  "kind": "rootline/explain",
  "path": "file.md",
  "stem_chain": [".stem", "../.stem"],
  "fields": [
    {
      "name": "estado",
      "value": "Pending",
      "origin": "frontmatter|derived|aggregated",
      "source": ".stem path",
      "expression": "expr (if derived/aggregated)"
    }
  ],
  "errors": []
}
```

Table: Field, Value, Origin, Source, Expression columns.

Use `explain` when the user needs to understand:
- Why a field has a specific value
- Which `.stem` file defines a field
- How derived or aggregated fields are computed
- What the `.stem` inheritance chain is for a file
