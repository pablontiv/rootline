---
name: rootline
description: |
  Rootline CLI operations: validate documents, fix errors, describe schemas,
  scaffold new documents, query records, view tree/stats, explain field derivation,
  build dependency graphs, migrate schemas, and infer schemas from existing files.
  Use when the user says "validate", "fix", "describe", "new doc", "query",
  "tree", "stats", "explain", "graph", "migrate", "init", "rootline",
  "verificar", "crear documento", "que campos necesito", "buscar registros".
argument-hint: "[command] [args...]"
allowed-tools: Bash, Read, AskUserQuestion
---

# /rootline — CLI Operations

Rootline treats the filesystem as a database. This skill wraps all rootline CLI commands.

## Command Reference

| Command | Purpose | When to use |
|---------|---------|-------------|
| `validate` | Check documents against `.stem` schemas | User wants to verify files are correct |
| `fix` | Auto-repair validation errors | After validate finds fixable errors |
| `describe` | Show effective schema for a directory | User asks what fields are needed |
| `new` | Scaffold a document with frontmatter | User wants to create a new file |
| `query` | Search and filter records | User wants to find records by field values |
| `tree` | Hierarchical view with completion counts | User wants to see project structure |
| `stats` | Summary counts by type and state | User wants aggregate numbers |
| `explain` | Trace how a field got its value | User needs to debug derived/aggregated fields |
| `graph` | Dependency graph from wiki-links | User wants to visualize dependencies |
| `migrate` | Schema change detection and migration | User modified `.stem` and needs to adapt |
| `init` | Infer `.stem` from existing documents | User has docs but no schema yet |

## Global Flags

All commands support:

- `--output json|table` (`-o`) — output format (default: `json`)
- `--field <dot-path>` — extract specific field from JSON output (repeatable)

Several transversal commands also support:

- `--where "expr"` — filter with expr-lang syntax (`==`, `!=`, `in`, `contains`, `exists`, `&&`)

## Routing

Based on user intent, apply the appropriate command group. Read the linked reference for detailed procedures, flags, and output formats.

### Validation (validate + fix)

Validate documents against `.stem` schemas and auto-fix errors.

```bash
rootline validate [file...] --output json          # Single or multiple files
rootline validate --all                             # All files in scope
rootline validate --all --where "estado == 'Pending'" --strict
rootline validate --staged                          # Git staging area only
rootline fix [file...] --dry-run                    # Preview fixes
rootline fix --all                                  # Fix all fixable errors
```

Key flags: `--all`, `--staged`, `--strict`, `--where`

**Procedure**: Determine target (file, directory, or all) → run validate → parse JSON → present errors with context → suggest `rootline fix` for fixable issues.

For detailed procedures and JSON output formats, see [ref-validate.md](ref-validate.md).

### Schema Inspection (describe + new)

Inspect effective schemas and scaffold new documents.

```bash
rootline describe <path> --output json              # Show merged schema
rootline describe <dir> --field schema.id.next      # Get next auto-ID
rootline new <filepath> --dry-run                   # Preview scaffold
rootline new <filepath>                             # Create document
```

Key flags: `--field` (extraction), `--dry-run`, `--force`

**Procedure**: For describe — show schema as table with field/type/required/values/source. For new — get next ID via describe, preview with dry-run, create, then validate.

For detailed procedures and JSON output formats, see [ref-schema.md](ref-schema.md).

### Queries & Exploration (query + tree + stats + explain)

Search, explore, and understand the document database.

```bash
rootline query --where "estado == 'Pending'" --limit 10
rootline query --count                              # Just the count
rootline tree --where "estado != 'Completed'"       # Filtered tree
rootline stats --from docs/epics/                   # Summary counts
rootline explain <file>                             # Trace field derivation
```

Key flags: `--where`, `--count`, `--limit`, `--from`

**Procedure**: Choose the right command based on what the user needs — query for filtering records, tree for structure, stats for aggregates, explain for debugging field values.

For detailed procedures and JSON output formats, see [ref-query.md](ref-query.md).

### Advanced Operations (graph + migrate + init)

Dependency graphs, schema migrations, and schema inference.

```bash
rootline graph --format mermaid                     # Dependency visualization
rootline graph --check                              # Validate links (cycles + broken)
rootline migrate --dry-run                          # Detect schema changes
rootline migrate --rename old=new                   # Rename fields
rootline migrate --split                            # Flat → hierarchical .stem
rootline init --dry-run                             # Infer schema from docs
```

These commands are used less frequently. For detailed procedures, see [ref-advanced.md](ref-advanced.md).

## Output Handling

1. Always use `--output json` when parsing results programmatically
2. All JSON responses include `"version": 1` and `"kind": "rootline/<command>"`
3. Use `--output table` when displaying results directly to the user
4. Use `--field` for extracting specific values (e.g., `--field schema.id.next`)
