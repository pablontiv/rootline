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
    field: blocked_by             # variable injected into derive expressions
    value_field: estado           # target-record field collected for that variable
  reference:
    target: ".*"                  # any target allowed
```

Without a link schema, all link types are allowed by default.

## CLI Usage

```bash
rootline graph docs/epics/                             # Dependency graph as JSON (default)
rootline graph docs/epics/ --format mermaid -o table   # Mermaid diagram
rootline graph docs/epics/ --format dot -o table       # Graphviz DOT diagram
rootline graph docs/epics/ --check                     # Validate only: cycles + broken links (text + exit code)
```

### Flags

| Flag | Description |
|------|-------------|
| `--format dot\|mermaid` | Diagram format when `-o table` (default: `dot`) |
| `--check` | Validate only — reports cycles and broken links, no diagram |
| `--fail-cycles` | Treat cycles as check failures; exit with code 1 if any found (default: cycles are informational) |
| `--quiet-cycles` | Suppress per-cycle enumeration when informational (only affects output when cycles are detected and not failing) |
| `--where "expr"` | Filter records before building graph (expr-lang syntax, repeatable) |

### Link Styles

Rootline can extract `[[wiki-links]]` and `[markdown](links)` reference styles. The styles recognized in graph building are configured in `.stem` via the `links.styles` field; when omitted, the effective set is `[wikilink]`.

To control which link styles are processed in the graph:

```yaml
# .stem file
links:
  styles: [wikilink, markdown]  # Replacement set; include both to recognize both styles
  checks:
    resolve: true              # Check that targets resolve (case-sensitive)
    anchors: true              # Validate heading anchors
    encoding: true             # Check for `%20` encoding in URLs
    cycles: true               # Fail graph --check if cycles found (optional)
```

`links.styles` replaces the default set; it does not add to it. For example, `styles: [markdown]` recognizes markdown links only and stops recognizing wikilinks until `wikilink` is listed as well. The graph command respects this setting.

### With --where

Filter records before building the dependency graph:

```bash
rootline graph docs/epics/ --where 'tipo != "feature"'           # Exclude features
rootline graph docs/epics/ --where 'estado == "Specified"' --check  # Check only specified tasks
```

### With --check

`graph` and `query` scan the same record set `validate --all` does: `scope.match` and
`.stemignore` both apply. A file the schema declares out of governance is not a node, contributes
no edges, and cannot fail `--check`. A tree carrying no schema at all is still graphed.

Typed link rules and `links.checks.cycles` resolve **per record**, like link styles. Hardening
declared in a subdirectory used to evaporate when the command ran from the repository root — the
most likely CI invocation. A cycle now fails the check when any node in it is governed by a schema
that asked for it, leaving unrelated cycles in never-opted-in subtrees informational.

### How a link target resolves

`graph` resolves links through the same engine `validate` uses, so they agree when basename fallback is off. Wikilinks infer `.md` (`[[b]]`→`b.md`, `[[sub/README]]`→`sub/README.md`), markdown targets resolve literally, and `/x.md` resolves against the scan root. A path-less target matches a uniquely named record anywhere only when `links.basename_fallback: true`; `graph` and query traversal have the full index needed for that lookup, while `validate` reports `link_unverifiable` instead of guessing. A resolved target is never reported broken **even when the schema does not govern it** — `scope.match` and `.stemignore` declare what is *governed*, not what *exists*, so such a link is an edge to a non-node. Unresolved targets are reported verbatim.

When the effective `.stem` declares `links.checks`, `--check` also reports the anchor and
encoding failures `validate` reports, so a green `graph --check` now means what it looks like it
means. Resolution failures stay under "Broken links" rather than being listed twice. Checks are
opt-in: a schema declaring none reports nothing extra.

Prints a text report and exits 1 when a blocking problem is found (broken links always block; cycles block only with `--fail-cycles` or `links.checks.cycles: true`, otherwise they are informational):

```text
Broken links: 1
  T002-task.md:7 → T999-nonexistent (blocks) — did you mean: T005-deploy.md?
```

When nothing is wrong it prints `No cycles or broken links found.` and exits 0. `--check` is a text-plus-exit-code validator; it does not emit JSON — use the default JSON mode below to extract link data.

Passing `--output` explicitly alongside `--check` is an error, not a no-op: the flag could never have been honoured, and being told so beats a pipeline discovering it downstream. The default `-o json` is not an explicit request, so `rootline graph docs/ --check` is unaffected. See [Output Formats](output.md).

### Output Ordering

Every ordered array `graph` emits is a function of the graph alone — never of Go map iteration
or of the order files were scanned — so output is byte-stable across runs and safe to commit,
diff, or assert on in CI:

| Array | Order |
|-------|-------|
| `nodes` | lexical by path |
| `edges` | `(source, target, line, type)` |
| `cycles` | each cycle rotated to start at its lexicographically smallest member and closed by repeating it; the list sorted element-wise, shorter first on a tie |
| `broken_links` | `(source, target, line, type)` |
| `broken_links[].suggestions` | ascending Levenshtein distance, then lexical by path; at most three |

The `--check` numbered enumeration prints the same `cycles` order and its broken-link report uses
the same `broken_links` and suggestion order, so every line is stable for the same input.

A cycle through `b.md → c.md → a.md → b.md` therefore always prints as
`a.md → b.md → c.md → a.md`: rotating a directed cycle does not change which links it contains,
so one rotation is picked as the printed representation.

**Cycle counting caveat.** Cycle detection reports back edges found over a canonical spanning
forest, not an exhaustive enumeration of every elementary circuit. Cycles that overlap are
represented, not individually listed. `cycles` is stable and reproducible, but read its length as
a count of detected back edges, not as the number of distinct circuits in the graph.

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

# Extract broken-link details (default JSON mode — not --check)
rootline graph docs/ --field 'broken_links'
# [{"source": "T002-task.md", "target": "T999-nonexistent", "type": "blocks", ...}]

# Extract simple broken link targets
rootline graph docs/ --field 'broken_links[].target'
# ["T999-nonexistent"]
```

The `--field` flag applies to graph JSON output and supports:
- Simple paths: `nodes`, `edges`, `cycles`, `broken_links`
- Array projection: `edges[].source`, `edges[].target`, `broken_links[].target`

`--field` requires `-o json`; combining it with `-o table` (the diagram) is an error rather than a silent no-op. It is repeatable — several paths yield a JSON array in flag order. See [Output Formats](output.md).

## Target Resolution

Links resolve targets by:
1. Relative path from the source document's directory.
2. Root-anchored path (`/x.md`) against the scan root.
3. Optional basename fallback — when `links.basename_fallback: true`, `graph` and query traversal can match a path-less target to one unique record in the scanned tree. The default is off.
