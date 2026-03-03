# Advanced Operations Reference

## graph — Dependency graph from wiki-links

### Usage

```bash
rootline graph                              # DOT format (default)
rootline graph --format mermaid             # Mermaid format
rootline graph --check                      # Validate only (cycles + broken links)
rootline graph --where "tipo == 'feature'"  # Filtered
```

### Flags

| Flag | Description |
|------|-------------|
| `--format dot\|mermaid` | Output format (default: `dot`) |
| `--check` | Validate only, no diagram output |
| `--where "expr"` | Filter expression (repeatable) |

### Output format (`kind: "rootline/graph"`):

```json
{
  "version": 1,
  "kind": "rootline/graph",
  "nodes": [...],
  "edges": [...],
  "cycles": [],
  "broken_links": []
}
```

- **DOT**: Graphviz digraph syntax (`rankdir=LR`)
- **Mermaid**: `graph TD` syntax with sanitized node IDs
- **Check mode**: Text report of cycles and broken links, exit code 1 if problems found

Links are filtered by schema — only structurally relevant link types are included.

---

## migrate — Schema change detection and migration

### Usage

```bash
rootline migrate                            # Diff current .stem vs git HEAD
rootline migrate .stem --from old.stem      # Diff against specific file
rootline migrate --rename old_field=new     # Rename a field across all docs
rootline migrate --split                    # Flat .stem → hierarchical files
rootline migrate --to-v2                    # Upgrade version field to v2
rootline migrate .stem --from-levels        # Convert v1 levels: to v2 match:
rootline migrate --dry-run <flag>           # Preview any migration
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Report changes without modifying |
| `--from <file>` | Compare against specified .stem file |
| `--rename old=new` | Rename a field in all documents and .stem files |
| `--split` | Split flat .stem into hierarchical per-level files |
| `--to-v2` | Upgrade .stem version from 0/1 to 2 |
| `--from-levels` | Convert v1 `levels:` syntax to v2 `match:`-based fields |

### Output formats

- **Diff mode** (`kind: "rootline/migrate-diff"`): Changes with field, kind, breaking flag, message
- **Rename mode** (`kind: "rootline/migrate-rename"`): Files and stems updated, migration log appended
- **Split mode**: Creates hierarchical .stem files with auto-generated aggregates
- **To-v2 mode** (`kind: "rootline/migrate-to-v2"`): Files upgraded, summary
- **From-levels mode**: Prints converted .stem content to stdout

### Typical workflow

```bash
# After modifying .stem
rootline migrate --dry-run                  # See what changed
rootline migrate                            # Apply migration

# Rename a field
rootline migrate --rename estado=status --dry-run
rootline migrate --rename estado=status

# Upgrade schema format
rootline migrate --to-v2 --dry-run
rootline migrate --to-v2
```

---

## init — Infer schema from existing documents

### Usage

```bash
rootline init                               # Current directory
rootline init <path> --dry-run              # Preview inferred schema
rootline init <path> --force                # Overwrite existing .stem
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Print inferred .stem to stdout without writing |
| `--force` | Overwrite existing .stem file |

### Behavior

- Analyzes frontmatter across all documents in the directory
- Detects field types, enum values, required fields
- **Hierarchy detection**: Recognizes naming patterns (E##, F##, S###, T###)
  - Flat mode: Single `.stem` with inferred schema
  - Hierarchical mode: Single `.stem` with `version: 2`, match-based per-level schema, auto-generated aggregates

### Typical workflow

```bash
rootline init docs/epics/ --dry-run         # Preview inferred schema
rootline init docs/epics/                   # Write .stem
rootline validate --all docs/epics/         # Verify against new schema
```
