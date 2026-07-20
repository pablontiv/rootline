# Rootline

[![CI](https://github.com/pablontiv/rootline/actions/workflows/ci.yml/badge.svg)](https://github.com/pablontiv/rootline/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: PolyForm Noncommercial](https://img.shields.io/badge/License-PolyForm%20Noncommercial-blue)](LICENSE)

## Rootline turns Markdown into a governed, queryable knowledge system.

Keep your Markdown consistent, connected, and queryable as it grows.

Rootline treats your documentation as structured data. **`.stem` files define schemas** (what fields must exist, their types, allowed values). **Validation rules** enforce consistency. **Queries** retrieve relevant records without reading every file. **Field provenance** shows where every value came from and how it was computed. Humans, automation, and AI agents all consume the same governed outputs.

---

## Table of Contents

- [Installation](#installation)
- [Why It Matters](#why-it-matters)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Command Capabilities](#command-capabilities)
- [Optional Integrations](#optional-integrations)
- [Documentation & References](#documentation--references)
- [Development](#development)
- [License](#license)

---

## Installation

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/pablontiv/rootline/master/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/pablontiv/rootline/master/install.ps1 | iex
```

### From source

```bash
go install github.com/pablontiv/rootline/cmd/rootline@latest
```

---

## Why It Matters

As Markdown documentation grows, two problems emerge: **structural drift** and **retrieval difficulty**. Rootline solves both.

- **Schema inheritance prevents drift**: Parent directories define rules that child documents inherit. Fields, types, and constraints flow top-down; mutations raise errors immediately.
- **Validation rules catch inconsistencies early**: `required`, `enum`, and structural constraints enforce consistency without manual review.
- **Queryable fields eliminate document reading**: Search by metadata (estado, tipo, tags) using declarative filters — no grep, no manual scanning.
- **Link graphs show dependencies**: Discover which documents reference each other, block on external targets, or form cycles.
- **Humans, agents, and automation consume the same governed outputs**: Stable JSON contracts mean your Markdown structure is machine-readable and auditable.

Rootline does not render documentation. It **models** it — making your Markdown a queryable, governed knowledge system.

---

## Quick Start

Initialize your documentation directory, validate its structure, and query its contents.

```bash
# 1. Initialize — infer .stem schema from existing documents
rootline init docs/

# 2. Query — find records by metadata
rootline query docs/ --where 'estado == "published"'

# 3. Validate — check documents against their schema rules (requires Git tree)
# Note: Schema discovery walks up the directory tree until it finds .git
git init docs/  # if not already a git repo
rootline validate --all docs/

# 4. Graph — visualize document dependencies
rootline graph docs/

# 5. New — scaffold a document from the effective schema
rootline new docs/task-001.md

# 6. Explain — trace where a field value came from
rootline explain docs/task-001.md
```

All commands output JSON by default. Use `--output table` for human-readable tables.

---

## Core Concepts

### What Rootline Models

- **Directory hierarchy**: Directories are tables; files are records.
- **Inherited rules**: `.stem` files define schemas; rules flow from parent to child via top-down merge.
- **Derived fields**: Expressions compute fields (slugify, concatenate, filter); aggregates roll up from children to parents.
- **Links**: Documents reference each other via `[[wiki-links]]` and `[markdown](links)`, forming a queryable dependency graph.
- **Queryable outputs**: Every command returns stable JSON; records can be filtered, sorted, and projected.

### How Schema Inheritance Works

Rootline discovers schemas by **walking up** your directory tree:

1. Start at the target path (file or directory)
2. Collect `.stem` files at each level, moving up toward the filesystem root
3. Stop at `.git` directory (repository boundary)
4. Merge collected schemas from root to leaf (parent → child)

Each level can add new fields, override parent definitions, or remove inherited rules (with `null`). Type-driven merge rules ensure predictable inheritance (maps merge key-level; arrays and scalars replace entirely).

**Git is required for schema discovery** because the `.git` boundary marks repository scope. Outside a Git tree, schema resolution fails.

### What Validation Means

Validation enforces consistency using rules defined in `.stem`:

- **required**: Field must be present and non-empty
- **enum**: Field value must be one of allowed choices
- **type**: Field must match declared type (string, number, section, array)
- **exists**: Path (file or directory) must exist
- **structural**: Directory naming, required children, index files

Violations are reported as errors; the command exits with code 1. Use `--strict` to treat warnings as errors.

### What Queryability Means

Queries retrieve relevant records without scanning entire documents:

- Declarative filtering: `--where 'estado == "published" && tipo == "epic"'`
- Metadata projection: `--select path,estado,title` returns compact rows
- Graph traversal: `--has-inbound` / `--has-outbound` with link predicates
- Counted results: `--count` returns summary statistics

All outputs are JSON, suitable for piping to automation and AI consumers.

---

## The `.stem` File — Your DDL

A `.stem` file is the DDL schema for a directory. It defines what fields exist, what types they have, which are required, and how values are validated. Rootline resolves schemas using **walk-up discovery + top-down merge**:

1. From the target path, walk **up** collecting `.stem` files until `.git` root
2. Merge them **top-down** (parent → child)

### Merge Rules

| YAML type | Behavior | Example |
|-----------|----------|---------|
| **map** | Key-level merge | Child adds or overrides keys |
| **array** | Replace | Child redefines entirely |
| **scalar** | Replace | Child overrides value |
| **null** | Remove | Child removes inherited key |

### Example

```yaml
version: 2

schema:
  title: { type: string, required: true }
  status:
    type: enum
    values: [draft, review, published]
    default: draft
    required: true
  ejecutable_en: { type: string, required: true, match: "T*" }
  "## Summary": { type: section, required: true }
  "## Changelog": { type: section, default: "<!-- TODO -->" }

aggregate:
  completed: 'len(filter(descendants, .status == "published"))'

links:
  allowed: [blocks, depends]
```

> Sections (`type: section`) are first-class schema fields — validated, defaulted, and queryable alongside frontmatter.

---

## CLI

Rootline ships as a **single static Go binary** with no dependencies.

> **Universal Filtering**: Most commands support `--where 'expr'` (expr-lang syntax) to filter records before processing.

```bash
# Core
rootline validate [file|--all|--staged] [--where 'expr'] [--strict]  # Check documents against .stem rules
rootline query [path] --where 'expr' [--count] [--limit N]  # Search by metadata (expr-lang syntax)
rootline describe <path>                  # Show effective schema for a directory
rootline tree [path] [--where 'expr']     # Hierarchical view with completion counts
rootline stats [path] [--where 'expr']    # Summary counts by estado and tipo
rootline graph [path] [--where 'expr']    # Dependency graph (DOT, Mermaid, --check, --open)
rootline explain <file>                   # Trace field origins, derivations, and errors

# Document lifecycle
rootline init [path] [--force] [--template owner/repo]  # Infer .stem or fetch from remote
rootline new <file> [--force] [--dry-run] # Scaffold document from effective schema
rootline set <file> field=value [...]     # Mutate frontmatter and sections with validation
rootline fix [file|--all]                 # Auto-repair: add fields, fix enums, propose changes
rootline validate --all --where 'expr'   # Validate only records matching filter
rootline migrate [path]                   # Detect schema changes, rename, split, --to-v2, --from-levels
rootline analyze [path] [--incremental]   # Run 14 detectors (12 data + 2 governance), produce report
rootline schema apply --report <file>     # Apply schema proposals to .stem (accepts analyze reports for schema changes)
rootline repair apply --report <file>     # Apply data-only repairs to document frontmatter

# Tooling
rootline hooks install|uninstall|status   # Git pre-commit hook management
rootline completion bash|zsh|fish         # Shell completion scripts
```

All commands support `--output json|table` and `--field` for dot-path extraction, including array projection:

```bash
# Dot-path extraction
rootline describe docs/prd/ --field schema.id.next
# "T004"

# Simple field projection from query
rootline query --where 'estado == "Pending"' --field path
# docs/projects/P01/tasks/T005-deploy-grafana.md

# Array projection: extract fields from arrays (rows, edges, etc.)
rootline query --field 'rows[].path'
# ["docs/projects/P01/tasks/T005-deploy-grafana.md", ...]

rootline query --field 'rows[].frontmatter.estado'
# ["Pending", "In Progress", ...]

rootline graph docs/ --field 'edges[].source'
# ["docs/api/auth.md", ...]

# Compact query projections with --select (JSON, JSONL, CSV)
rootline query --select path,estado,title
# {"rows": [{"path": "...", "estado": "Pending", "title": "..."}, ...]}

rootline query --select path,estado --output jsonl
# {"path": "...", "estado": "Pending"}
# {"path": "...", "estado": "In Progress"}

rootline query --select path,estado,title --output csv
# path,estado,title
# docs/api/auth.md,Pending,Authentication Guide
# docs/api/endpoints.md,In Progress,Endpoint Reference

# Filtering across commands
rootline tree docs/epics/ --where 'estado != "Completed"'
rootline stats docs/epics/ --where 'tipo == "software-module"'
rootline graph docs/epics/ --where 'tipo != "feature"' --check
```

### Query Expressions

Queries use [expr-lang/expr](https://expr-lang.org/) syntax. Multiple `--where` flags are combined with AND:

```bash
rootline query --where 'estado == "Pending"'
rootline query --where 'tipo in ["lxc", "vm"]' --where 'estado != "Completed"'
rootline query --where 'body contains "migration"'
rootline query --where 'tags != nil' --count
```

### Derivation & Aggregation

`.stem` files can define derived fields (computed per-record) and aggregates (rolled up from children to parent index files):

```yaml
derive:
  slug: 'slugify(titulo)'
  name_lower: 'lower(nombre)'

aggregate:
  total: 'len(descendants)'
  completed: 'len(filter(descendants, .estado == "Completed"))'
```

Derived and aggregated fields appear alongside frontmatter in query results, stats, and tree output.

### Dependency Graph

Documents reference each other via `[[wiki-links]]` in their body. Rootline extracts these links and builds a directed graph:

```bash
rootline graph docs/ --format mermaid   # Mermaid diagram
rootline graph docs/ --format dot       # Graphviz DOT
rootline graph docs/ --check            # Validate: detect cycles and broken links
rootline graph docs/ --open             # Open interactive diagram in browser
```

Link schemas in `.stem` files control which link types are allowed and validate targets against regex patterns.

### Fix & Proposals

`rootline fix` goes beyond adding missing fields — it proposes intelligent repairs:

```bash
rootline fix doc.md --dry-run    # Preview proposed changes
rootline fix --all               # Fix all files in scope
```

Proposals include: correct misspelled enum values (Levenshtein matching), extend `.stem` enums for new valid values, migrate values with wiki-link insertion, and infer fields from child documents.

### Explain

`rootline explain` traces **why** a document has its current state — field origins, derivation expressions, aggregation sources, and validation errors:

```bash
rootline explain docs/projects/P01/F01/README.md
```

Shows each field's origin (frontmatter, schema default, derived, or aggregated) with the source `.stem` file and expression.

---

## AI-Native

Rootline is designed as a **structured knowledge source for AI assistants**. All commands output stable JSON with `"version": 1` contracts, making them suitable for tool use and automation.

### CLI-first automation

AI assistants and automation should call the Rootline CLI directly and consume stable JSON output from commands such as `query`, `validate`, `describe`, `tree`, `stats`, `explain`, `fix`, `graph`, `set`, `trace`, and `new`.

### Engine vs. agent: division of labor

Rootline's engine decides everything resolvable **from form** — frequency
thresholds (a field present in ≥80% of records is `required`), unanimous or
majority value agreement, and structural conventions (directory naming, type
consistency). Decisions that need **meaning** — is this value semantically the
same as that one? — are not guessed: `analyze` marks those proposals
`requires_agent` for a human or agent to resolve. The report exposes
**percentage evidence, not opinions**; consumers apply their own thresholds.

---

## Documentation

| Topic | Description |
|-------|-------------|
| [Init](docs/init.md) | Schema inference from existing documents |
| [Validate](docs/validate.md) | Validation rules, batch mode, staged checks |
| [Describe](docs/describe.md) | Describe output, field extraction, source tracking |
| [Query Engine](docs/query.md) | Query contract, operators, result shapes |
| [New](docs/new.md) | Document scaffolding from effective schema |
| [Set](docs/set.md) | Mutate frontmatter and sections with schema validation |
| [Fix & Proposals](docs/fix.md) | Auto-repair, enum correction, field inference |
| [Explain](docs/explain.md) | Field origin tracing, derivation chain, error diagnosis |
| [Tree](docs/tree.md) | Hierarchical view with completion counts |
| [Stats](docs/stats.md) | Summary counts by type and state |
| [Dependency Graph](docs/graph.md) | Wiki-links, link schema, cycle detection, DOT/Mermaid |
| [Derivation Engine](docs/derivation.md) | Derive and aggregate expressions, builtins, linked fields |
| [Schema Migration](docs/migrate.md) | Breaking change detection, field rename, v2 upgrade |
| [Levels & Match](docs/levels.md) | Hierarchical field scoping with match patterns |
| [Extensibility](docs/extensibility.md) | Extractor architecture, future formats |
| [Visual Identity](docs/identity.md) | Logo, colors, usage guidelines |

---

## Updating

Release builds auto-update in the background using a staged async pattern — the new binary is downloaded during run N and applied at the start of run N+1. Local builds (`version == "dev"`) skip this entirely. See [docs/auto-update.md](docs/auto-update.md) for details.

---

## Development

```bash
go build ./cmd/rootline/          # Build
go test ./... -race               # Tests with race detector
go vet ./...                      # Static analysis
golangci-lint run ./...           # Full lint
```

Pre-commit hooks run `golangci-lint` and `gofmt` automatically. Commits follow [Conventional Commits](https://www.conventionalcommits.org/) (`type(scope): description`), enforced by a commit-msg hook. To manually sync skills and rebuild after pulling: `bash .githooks/pre-push`.

---

## License

[PolyForm Noncommercial 1.0.0](LICENSE) — free for non-commercial use.
