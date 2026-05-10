---
estado: Completed
---
# Tree View

`rootline tree` displays the document hierarchy with completion counts derived from `estado` fields. Leaf nodes count as completed when `estado == "Completed"`; directory nodes sum their children.

## CLI Usage

```bash
rootline tree                                     # Current directory
rootline tree docs/epics/                         # Specific path
rootline tree --where 'estado != "Completed"'     # Filtered
rootline tree -o json                             # JSON output
```

### Flags

| Flag | Description |
|------|-------------|
| `--where "expr"` | Filter records before building tree (expr-lang, repeatable) |

## Table Output

```
docs [10/20]
├── derivation.md [Completed]
├── describe.md [Completed]
├── fix.md [Completed]
├── graph.md [Completed]
├── query.md [Completed]
└── research [0/10]
    ├── intrinsic-hierarchy-principle.md [In Progress]
    ├── kedral [0/4]
    │   ├── README.md [—]
    │   └── design.md [—]
    └── opportunity-areas.md [Pre-research]
```

Records without `estado` show `[—]`. Directories show `[completed/total]`.

## JSON Result

```json
{
  "version": 1,
  "kind": "rootline/tree",
  "root": {
    "name": "docs",
    "path": "docs",
    "completed": 10,
    "total": 21,
    "children": [
      {
        "name": "query.md",
        "path": "query.md",
        "completed": 1,
        "total": 1,
        "is_leaf": true,
        "estado": "Completed"
      },
      {
        "name": "research",
        "path": "research",
        "completed": 0,
        "total": 10,
        "children": [ ... ]
      }
    ]
  }
}
```
