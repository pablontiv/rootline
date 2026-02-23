---
estado: Completado
---
# Dependency Graph

Rootline extracts, validates, and visualizes relationships between documents via typed wiki-links.

## Wiki-Link Syntax

Links are declared inline in document body:

```markdown
[[blocks:T003-setup-database]]
[[depends:T001-schema-design]]
[[reference:architecture-doc]]
```

Format: `[[type:target]]` or `[[target]]` (untyped). Links inside fenced code blocks and inline code are ignored.

## Link Schema in .stem

```yaml
links:
  blocks:
    target: "^T\\d{3}-"           # target must match pattern
    fields:
      estado: estado              # inject linked field into derive expressions
  reference:
    target: ".*"                  # any target allowed
```

Without a link schema, all link types are allowed by default.

## CLI Usage

```bash
rootline graph docs/epics/                    # DOT output (default)
rootline graph docs/epics/ --format mermaid   # Mermaid output
rootline graph docs/epics/ --check            # Validate only: cycles + broken links
```

### Flags

| Flag | Description |
|------|-------------|
| `--format dot\|mermaid` | Output format (default: `dot`) |
| `--check` | Validate only — reports cycles and broken links, no diagram |
| `--where "expr"` | Filter records before building graph (expr-lang syntax, repeatable) |

### With --where

Filter records before building the dependency graph:

```bash
rootline graph docs/epics/ --where 'tipo != "feature"'           # Exclude features
rootline graph docs/epics/ --where 'estado == "Specified"' --check  # Check only specified tasks
```

### With --check

Returns exit code 1 if cycles or broken links are found:

```json
{
  "version": 1,
  "kind": "rootline/graph-check",
  "cycles": [["T001", "T003", "T001"]],
  "broken_links": [
    {"source": "T002-task.md", "target": "T999-nonexistent", "type": "blocks", "line": 7}
  ]
}
```

## Target Resolution

Links resolve targets by:
1. Relative path from source document's directory
2. Basename fallback — unique match across the scanned tree
