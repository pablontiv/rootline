---
name: rootline
description: |
  Wrapper del CLI de rootline: validate, fix, describe, new, query, tree, stats,
  explain, graph, migrate, init, analyze, apply. Usar siempre que el usuario
  trabaje con documentos gestionados por rootline o schemas .stem — incluso sin
  decir "rootline". Usar este skill siempre que el usuario quiera validar
  documentos, verificar frontmatter, consultar registros estructurados, crear
  archivos scaffolded, bootstrap de consistencia, o inspeccionar schemas —
  incluso si solo dice "verificar si este archivo está bien", "que campos
  necesito", "buscar registros", "inferir schema", "hacer consistente",
  "check if this file is valid", o "what fields does this need".
  (No para: descomposición en epics/stories = roadmap, investigación = hypothesize,
  sync de docs = update-docs, debugging de Go, operaciones YAML no-rootline.)
argument-hint: "[command] [args...]"
allowed-tools: Bash, Read, AskUserQuestion
---

# /rootline — CLI Operations

Rootline treats the filesystem as a database. This skill wraps all rootline CLI commands.

## Common Workflows

### Bootstrap Consistency (most common)

When a user has a folder with markdown files and wants rootline to make it consistent:

```bash
rootline init <path> --dry-run              # 1. Preview inferred schema
rootline init <path>                        # 2. Write .stem (add --force if exists)
rootline validate --all --output json       # 3. Check for errors
rootline fix --all --dry-run                # 4. Preview fixes (if errors found)
rootline fix --all                          # 5. Apply fixes
rootline validate --all                     # 6. Verify clean state
```

**Key considerations during bootstrap:**
- If `init` infers a field as `enum` but a parent `.stem` defines it as `string`, you'll get a `type-consistency` error. Resolution: remove the field from the local `.stem` and let it inherit from the parent.
- If `init` infers date-like fields as `enum`, change them to `type: string` — dates are open-ended, not a closed set.
- Always run `--dry-run` before writing to preview what will change.
- After init, show the user the effective schema (`rootline describe <path>`) so they see both local and inherited fields.

### Create → Validate Loop

When scaffolding a new document:

```bash
rootline describe <dir> --field schema.id.next   # Get next auto-ID
rootline new <filepath> --dry-run                 # Preview scaffold
rootline new <filepath>                           # Create document
rootline validate <filepath>                      # Verify it's valid
```

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
| `analyze` | Run all inference detectors | User wants a full analysis report of patterns |
| `apply` | Apply inference results to .stem | User wants to update .stem based on analysis |

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

### Analysis (analyze)

Run all inference detectors and produce a structured report.

```bash
rootline analyze <directory> --output json          # Full analysis report
rootline analyze <directory> --output table          # Summary table
rootline analyze <directory> --incremental           # Only uncovered inferences
rootline analyze <directory> --field summary         # Extract summary only
```

Key flags: `--incremental` (filter inferences covered by .stem)

**Procedure**: Scans directory, extracts records, runs 12 detector categories (field types, required fields, enums, constants, link types, back-refs, cross-refs, section patterns, invariants, sub-schemas, dependencies, traceability). Produces AnalyzeReport JSON with `version: 1`. With `--incremental`, filters out inferences already covered by the existing `.stem` schema.

### Apply (apply)

Apply inference results to .stem files and document frontmatter.

```bash
rootline analyze <directory> --output json | rootline apply   # Pipe from analyze
rootline apply report.json                                    # From file
rootline apply report.json --output table                     # Human-readable
rootline apply report.json --dry-run                          # Preview changes
```

Key flags: `--dry-run` (show changes without applying)

**Procedure**: Reads an analyze report, applies schema-modifying inferences (extend_enum, add_required, add_default, set_type) to the closest `.stem` file and data corrections (migrate_value, correct_value, add_field) to document frontmatter. Inferences with `requires_agent: true` are skipped with warnings.

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
