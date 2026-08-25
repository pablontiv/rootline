# Rootline

[![CI](https://github.com/pablontiv/rootline/actions/workflows/ci.yml/badge.svg)](https://github.com/pablontiv/rootline/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue)](LICENSE)

Keep your Markdown consistent, connected, and queryable as it grows.

Rootline treats your documentation as structured data. **`.stem` files define schemas** (what fields must exist, their types, allowed values). **Validation rules** enforce consistency. **Queries** retrieve relevant records without reading every file. **Field inspection** shows where frontmatter and source-backed values came from and how computed fields were produced. Humans, automation, and AI agents all consume the same governed outputs.

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
# 1. Initialize — infer a .stem schema from existing documents
# (writes a root: true marker so schema discovery knows where to stop)
rootline init docs/

# 2. Query — find records by metadata
rootline query docs/ --where 'estado == "published"'

# 3. Validate — check documents against their schema rules
# Schema discovery walks up the tree until it hits a root: true .stem
# (or the filesystem root). No Git required.
rootline validate --all docs/

# 4. Graph — render the dependency diagram (Mermaid)
rootline graph docs/ --format mermaid -o table

# 5. New — scaffold a document from the effective schema
rootline new docs/task-001.md

# 6. Explain — trace where a field value came from
rootline explain docs/task-001.md
```

Data and inspection commands output JSON by default when they have a machine-readable envelope. Use `--output table` for supported human-readable tables. (`graph --check` reports cycles and broken links as text plus an exit code.)

---

## Core Concepts

### What Rootline Models

- **Directory hierarchy**: Directories are tables; files are records.
- **Inherited rules**: `.stem` files define schemas; rules flow from parent to child via top-down merge.
- **Derived fields**: Expressions compute fields (slugify, concatenate, filter); aggregates roll up from children to parents.
- **Links**: Documents reference each other via `[[wiki-links]]` by default and `[markdown](links)` when `links.styles` includes `markdown`, forming a queryable dependency graph.
- **Queryable outputs**: Data and inspection commands return stable JSON; records can be filtered, sorted, and projected where the command supports those flags.

### How Schema Inheritance Works

Rootline discovers schemas by **walking up** your directory tree:

1. Start at the target path (file or directory)
2. Collect `.stem` files at each level, moving up until a `.stem` declares `root: true` — the governance boundary — or, if none does, until the filesystem root
3. Merge collected schemas from root to leaf (parent → child)

Each level can add new fields or narrow parent definitions. Type-driven merge rules ensure predictable inheritance (maps merge key-level; arrays and scalars replace entirely).

**A child never removes anything.** Not in `schema:`, not in `derive:`, not in `aggregate:`. Setting a key to `null` in a child does not delete it — a `.stem` that drops a parent's declaration silently reduces the guarantee that parent made to everything beneath it, which is the one thing hierarchical governance exists to prevent. If a field has to go, the structure is wrong: remove it from the `.stem` that declares it.

A `.stem` file with `root: true` marks the governance boundary; walk-up discovery stops there. The marker is required, not optional: if the walk reaches the filesystem root without finding one, the chain may have collected `.stem` files from outside your project, so the boundary preflight refuses to run governed commands. On a terminal it offers to add `root: true` for you; in a pipeline or CI it is a hard error, and the fix is to add the marker to the `.stem` at the top of your project. **Git is optional** — Rootline works in any directory, with or without Git, and never uses `.git` as a boundary.

### The `.stem` File

A `.stem` file is the DDL schema for a directory. It defines what fields exist, what types they have, which are required, and how values are validated.

```yaml
version: 2

schema:
  title: { type: string, required: true }
  status:
    type: enum
    values: [draft, review, published]
    default: draft
    required: true

aggregate:
  completed: 'len(filter(descendants, .status == "published"))'

links:
  allowed: [blocks, depends]
```

> Body sections are source-backed values: use a real type with `source: body.section["## Summary"]`; frontmatter remains an explicit override.

### What Validation Means

Validation enforces consistency using rules defined in `.stem`:

- **required**: Field presence; `""` and `[]` are present, while `non_empty` is separate
- **enum**: Field value must be one of the declared `values:`
- **type**: Strict string, list, enum, sequence, link, boolean, and integer conformance without coercion
- **exists**: `exists` checks presence of an effective field, including source-backed or derived values
- **structural**: Directory naming, required children, index files

Violations are reported as errors; the command exits with code 1. Use `--strict` to treat warnings as errors.

### What Queryability Means

Queries retrieve relevant records without scanning entire documents:

- Declarative filtering: `--where 'estado == "published" && tipo == "epic"'`
- Metadata projection: `--select path,estado` returns compact rows
- Graph traversal: `--has-inbound` / `--has-outbound` with link predicates
- Counted results: `--count` returns summary statistics

These outputs are JSON, suitable for piping to automation and AI consumers.

---

## Command Capabilities

Rootline ships as a **single static Go binary** with no dependencies. Commands are grouped by use case. Traversal/data commands such as `query`, `stats`, `tree`, `graph`, and `validate --all` support `--where 'expr'` (expr-lang syntax) to filter records before processing.

### Validate & Govern

Check documents against inherited schemas and trace field origins.

- **`validate`** — Check documents against `.stem` rules
  - `rootline validate [file...]` — Single file
  - `rootline validate --all [--where 'expr'] [--strict]` — All files in scope
  - `rootline validate --staged` — Git staging area only

- **`describe`** — Show effective schema for a directory
  - `rootline describe <path>` — Merged schema with all inherited rules

- **`explain`** — Trace field origins, derivations, and errors
  - `rootline explain <file>` — Inspect field origins and computed expressions

### Query & Traverse

Find records and visualize dependencies.

- **`query`** — Search by metadata using declarative filters
  - `rootline query [path] --where 'expr'` — Filter records
  - `rootline query --where 'expr' --count` — Summary count
  - `rootline query --where 'expr' --select path,estado` — Compact row output
  - `rootline query --where 'expr' --has-inbound '<sub-expr>'` — Records with inbound links
  - `rootline query --where 'expr' --has-outbound '<sub-expr>'` — Records with outbound links

- **`tree`** — Hierarchical view with recursive record totals
  - `rootline tree [path] [--where 'expr']` — Directory structure with metadata

- **`graph`** — Dependency graph from governed link styles
  - `rootline graph [path]` — Dependency graph as JSON (default)
  - `rootline graph [path] --format dot|mermaid -o table` — Render a diagram
  - `rootline graph [path] --check` — Validate cycles and broken links (text report + exit code)
  - `rootline graph [path] --fail-cycles` — Treat cycles as errors

### Build & Maintain

Create and update documents.

- **`init`** — Generate `.stem` schema from existing documents
  - `rootline init [path]` — Infer schema from frontmatter patterns
  - `rootline init [path] --template owner/repo[@tag]` — Fetch `.stem` from remote
  - `rootline init [path] --force` — Overwrite existing `.stem`
  - `rootline init [path] --dry-run` — Preview without writing

- **`new`** — Scaffold a document from effective schema
  - `rootline new <filepath>` — Create with frontmatter pre-populated
  - `rootline new <filepath> --force` — Overwrite an existing file
  - `rootline new <filepath> --dry-run` — Preview generated content

- **`set`** — Mutate frontmatter fields with validation
  - `rootline set <file> field=value [field2=value2 ...]` — Set fields
  - `rootline set <file> field=@file` — Load content from file
  - `rootline set <file> ... --dry-run` — Preview changes

- **`fix`** — Auto-repair validation errors
  - `rootline fix [file...]` — Fix single file
  - `rootline fix --all` — Fix all files in scope
  - `rootline fix --dry-run` — Preview proposed changes

### Analyze & Evolve

Analyze patterns and manage schema evolution.

- **`analyze`** — Run 14 inference detectors (12 data + 2 governance)
  - `rootline analyze [directory]` — Produce structured report
  - `rootline analyze [directory] --incremental` — Only inferences not covered by existing `.stem`

- **`schema`** — Schema operations
  - `rootline schema propose [directory]` — Generate schema proposals
  - `rootline schema apply --report <file>` — Apply schema proposals to `.stem` files

- **`repair`** — Apply data-only repairs from analyze report
  - `rootline repair apply --report <file>` — Fix frontmatter only (not `.stem`)
  - `rootline repair apply --report <file> --dry-run` — Preview repairs

- **`migrate`** — Detect and apply schema changes
  - `rootline migrate [path]` — Compare current `.stem` against git HEAD
  - `rootline migrate [path] --rename old_field=new_field` — Bulk field rename
  - `rootline migrate [path] --split` — Convert flat `.stem` to hierarchical per-level files
  - `rootline migrate [path] --scaffold` — Create missing required sections
  - `rootline migrate [path] --dry-run` — Preview without modifying

### Infrastructure

- **`completion`** — Generate shell completion scripts
  - `rootline completion bash|zsh|fish` — Load in your shell

- **`hooks`** — Git pre-commit hook management
  - `rootline hooks install` — Enable pre-commit validation
  - `rootline hooks status` — Check installation status
  - `rootline hooks uninstall` — Remove hook

Commands with JSON envelopes support `--output json` and `--field` for dot-path extraction; `query` also supports `jsonl` and `csv` with `--select`. Commands that emit only human text or write files reject `--field`:

```bash
# Dot-path extraction
rootline describe docs/prd/ --field schema.id.next
# "T004"

# Query field extraction uses the rows array
rootline query --where 'estado == "Pending"' --field 'rows[].path'
# ["docs/projects/P01/tasks/T005-deploy-grafana.md", ...]

# Array projection: extract fields from arrays (rows, edges, etc.)
rootline query --field 'rows[].path'
# ["docs/projects/P01/tasks/T005-deploy-grafana.md", ...]

rootline query --field 'rows[].frontmatter.estado'
# ["Pending", "In Progress", ...]

rootline graph docs/ --field 'edges[].source'
# ["docs/api/auth.md", ...]

# Compact query projections with --select (JSON, JSONL, CSV)
rootline query --select path,estado
# {"rows": [{"path": "...", "estado": "Pending"}, ...]}

rootline query --select path,estado --output jsonl
# {"path": "...", "estado": "Pending"}
# {"path": "...", "estado": "In Progress"}

rootline query --select path,estado --output csv
# path,estado
# docs/api/auth.md,Pending
# docs/api/endpoints.md,In Progress

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

Derived and aggregate fields appear in query results. In `tree` JSON they are merged into leaf nodes' `frontmatter`; directory nodes carry recursive `total` counts and child nodes, not frontmatter.

### Dependency Graph

Documents reference each other via `[[wiki-links]]` by default, or via both wikilinks and markdown links when `.stem` sets `links.styles: [wikilink, markdown]`. Rootline extracts the governed styles and builds a directed graph:

```bash
rootline graph docs/                            # Dependency graph as JSON (default)
rootline graph docs/ --format mermaid -o table  # Mermaid diagram
rootline graph docs/ --format dot -o table      # Graphviz DOT
rootline graph docs/ --check                    # Validate: cycles + broken links (text + exit code)
```

Link schemas in `.stem` files control which link types are allowed and validate targets against regex patterns.

### Fix & Proposals

`rootline fix` goes beyond adding missing fields — it proposes intelligent repairs:

```bash
rootline fix doc.md --dry-run    # Preview proposed changes
rootline fix --all               # Fix all files in scope
```

Proposals include: correct misspelled enum values (Levenshtein matching), withheld `.stem` enum-extension suggestions for review, migrate values with wiki-link insertion, and aggregate propagation when configured.

### Explain

`rootline explain` traces **why** a document has its current state — field origins, derivation expressions, aggregation sources, and validation errors:

```bash
rootline explain docs/projects/P01/F01/README.md
```

Shows each field's origin (`frontmatter`, `schema`, `derived`, or `aggregate`). Frontmatter/source-backed schema fields carry `defined_in` and logical `source` directives when available; computed derive/aggregate fields carry expressions and currently do not report `.stem` provenance.

---

## Optional Integrations

Git is optional. Rootline works in any directory, with or without version control.

### Git-based Workflows

Rootline integrates with Git for continuous validation and collaborative workflows:

- **Manual staged validation** — `rootline validate --staged` is available when you want to check staged Markdown files
- **CI validation** — GitHub Actions workflows in `.github/workflows/` run tests and repository checks
- **Diff-aware reviews** — Queries and validation support `--where` filters, making it easy to review focused record subsets

These workflows are optional enhancements, not product requirements. You can use Rootline without Git by running commands manually.

---

## AI-Native

Rootline is designed as a **structured knowledge source for AI assistants**. Data commands output stable, versioned JSON contracts (each payload carries its own `version` field), making them suitable for tool use and automation.

### CLI-first automation

AI assistants and automation should call the Rootline CLI directly and consume stable JSON output from commands that emit envelopes, such as `query`, `validate`, `describe`, `tree`, `stats`, `explain`, `fix --all --dry-run`, and `graph` JSON mode. Commands such as `set`, `new`, `init`, and `graph --check` emit human text or perform writes instead of a JSON envelope.

### Engine vs. agent: division of labor

Rootline's engine decides everything resolvable **from form** — frequency
thresholds (a field present in >80% of records is inferred as `required`), unanimous or
majority value agreement, and structural conventions (directory naming, type
consistency). Decisions that need **meaning** — is this value semantically the
same as that one? — are not guessed: `analyze` marks those proposals
`requires_agent` for a human or agent to resolve. The report exposes
**percentage evidence, not opinions**; consumers apply their own thresholds.

---

## Documentation & References

### User Documentation

| Topic | Description |
|-------|-------------|
| [Output Formats](docs/output.md) | The `--output` contract and which command supports which format |
| [Init](docs/init.md) | Schema inference from existing documents |
| [Validate](docs/validate.md) | Validation rules, batch mode, staged checks |
| [Describe](docs/describe.md) | Describe output, field extraction, source tracking |
| [Query Engine](docs/query.md) | Query contract, operators, result shapes |
| [New](docs/new.md) | Document scaffolding from effective schema |
| [Set](docs/set.md) | Mutate frontmatter overrides with schema validation |
| [Fix & Proposals](docs/fix.md) | Auto-repair, enum correction, field inference |
| [Analyze](docs/analyze.md) | Infer schemas and patterns from documents |
| [Explain](docs/explain.md) | Field origin tracing, derivation chain, error diagnosis |
| [Tree](docs/tree.md) | Hierarchical view with recursive record totals |
| [Stats](docs/stats.md) | Total record counts, optionally filtered |
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

### Requirements

- **Product requirement**: Go 1.26+
- **Contributor workflow**: Git (for pre-commit hooks, tests, CI)

### Contributor Setup

```bash
go build ./cmd/rootline/          # Build
go test ./... -race               # Tests with race detector
go vet ./...                      # Static analysis
golangci-lint run ./...           # Full lint
```

Pre-commit hooks run `golangci-lint` and `gofmt` automatically. Commits follow [Conventional Commits](https://www.conventionalcommits.org/) (`type(scope): description`), enforced by a commit-msg hook. Pre-push keeps validation and skill synchronization without installing an unmerged branch build. Run `just install` explicitly when you want to install the current checkout.

**Note**: Git workflow is contributor-only; Rootline itself does not require Git.

---

## License

[Apache License 2.0](LICENSE) — free for commercial and non-commercial use.
