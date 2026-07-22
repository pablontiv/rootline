---
estado: Completed
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
| `--fail-cycles` | Treat cycles as check failures; exit with code 1 if any found (default: cycles are informational) |
| `--quiet-cycles` | Suppress per-cycle enumeration when informational (only affects output when cycles are detected and not failing) |
| `--where "expr"` | Filter records before building graph (expr-lang syntax, repeatable) |

### Link Styles

Rootline extracts both `[[wiki-links]]` and `[markdown](links)` reference styles. The links recognized in graph building are configured in `.stem` via the `links.styles` field (default: `[wikilink]`).

To control which link styles are processed in the graph:

```yaml
# .stem file
links:
  styles: [wikilink, markdown]  # Recognize both wiki-links and markdown links
  checks:
    resolve: true              # Check that targets resolve (case-sensitive)
    anchors: true              # Validate heading anchors
    encoding: true             # Check for `%20` encoding in URLs
    cycles: true               # Fail graph --check if cycles found (optional)
```

With `links.styles`, only listed styles are extracted and validated. The graph command respects this setting.

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
    {"source": "T002-task.md", "target": "T999-nonexistent", "type": "blocks", "line": 7, "suggestions": ["T001-setup-database.md", "T005-deploy.md"]}
  ]
}
```

### With --field (Field Projection)

Extract specific fields from graph results with array-aware dot-path syntax:

```bash
# Extract all edge sources
rootline graph docs/ --field 'edges[].source'
# ["T001-task.md", "T002-task.md"]

# Extract edge targets
rootline graph docs/ --field 'edges[].target'

# Extract both source and target as objects
rootline graph docs/ --field 'edges[]'

# Check for broken links and extract details
rootline graph docs/ --check --field 'broken_links'
# [{"source": "T002-task.md", "target": "T999-nonexistent", "type": "blocks", ...}]

# Extract simple broken link targets
rootline graph docs/ --check --field 'broken_links[].target'
# ["T999-nonexistent"]
```

The `--field` flag applies to graph JSON output and supports:
- Simple paths: `nodes`, `edges`, `cycles`, `broken_links`
- Array projection: `edges[].source`, `edges[].target`, `broken_links[].target`

## Target Resolution

Links resolve targets by:
1. Relative path from source document's directory
2. Basename fallback — unique match across the scanned tree
