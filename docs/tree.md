---
estado: Completed
---
# Tree View

`rootline tree` displays the document hierarchy with recursive record totals. Leaf nodes represent Markdown records; directory nodes sum descendant record counts.

## CLI Usage

```bash
rootline tree                                     # Current directory
rootline tree docs/epics/                         # Specific path
rootline tree --where 'estado != "Completed"'     # Filtered
rootline tree -o json                             # JSON output (default)
rootline tree -o table                            # ASCII tree
```

### Flags

| Flag | Description |
|------|-------------|
| `--where "expr"` | Filter records before building tree (expr-lang, repeatable) |

## Table Output

```
docs [21]
├── derivation.md [Completed]
├── describe.md [Completed]
├── fix.md [Completed]
├── graph.md [Completed]
├── query.md [Completed]
└── research [10]
    ├── intrinsic-hierarchy-principle.md [In Progress]
    ├── kedral [4]
    │   ├── README.md [—]
    │   └── design.md [—]
    └── opportunity-areas.md [Pre-research]
```

Records without the lifecycle/status field selected from the schema show `[—]`. Directories show `[total]`, where `total` is the recursive record count.

## JSON Result

```json
{
  "version": 2,
  "kind": "rootline/tree",
  "root": {
    "name": "docs",
    "path": "docs",
    "total": 21,
    "children": [
      {
        "name": "query.md",
        "path": "query.md",
        "total": 1,
        "is_leaf": true,
        "frontmatter": {
          "estado": "Completed"
        }
      },
      {
        "name": "research",
        "path": "research",
        "total": 10,
        "children": [ ... ]
      }
    ]
  }
}
```

The `total` count includes all children recursively; leaf nodes have `total: 1`. Leaf field values appear in the `frontmatter` object, with derived and aggregate values merged into that map. Directory nodes do not carry `frontmatter`.

`tree` supports `-o json` and `-o table` only. `-o jsonl` and `-o csv` are rejected — see [Output Formats](output.md).
