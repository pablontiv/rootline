# Rootline

[![CI](https://github.com/pablontiv/rootline/actions/workflows/ci.yml/badge.svg)](https://github.com/pablontiv/rootline/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: PolyForm Noncommercial](https://img.shields.io/badge/License-PolyForm%20Noncommercial-blue)](LICENSE)

A **file-based database and constraint engine** for structured documentation.

Rootline treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` files.

> **Status**: Engine and MCP server complete — all CLI commands and 8 MCP tools functional.

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Idea](#core-idea)
- [The `.stem` File](#the-stem-file)
- [CLI](#cli)
- [AI-Native](#ai-native)
- [Documentation](#documentation)
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

## Quick Start

```bash
# 1. Initialize — infer .stem rules from existing documents
rootline init docs/

# 2. Validate — check all documents against their rules
rootline validate --all

# 3. Describe — see what a valid document looks like
rootline describe docs/api/

# 4. Query — find documents by metadata (expr-lang syntax)
rootline query --where 'estado == "published"'

# 5. Scaffold — create a new document from the schema
rootline new docs/api/auth.md

# 6. Explain — trace why a field has a given value
rootline explain docs/api/auth.md

# 7. Graph — visualize document dependencies
rootline graph docs/ --check
```

---

## Core Idea

Documentation already has structure. Rootline makes it **explicit**, **inherited**, and **queryable**.

- The directory tree defines hierarchy
- Rules flow from parent to child via `.stem` files
- Fields are derived via expressions; aggregates roll up from children to parents
- Documents link to each other via `[[wiki-links]]`, forming a dependency graph
- All output is stable JSON, suitable for CI, automation, and AI

Rootline does not render documentation. It **models** it.

---

## The `.stem` File

A `.stem` file may appear in any directory. Rootline resolves configuration using **walk-up discovery + top-down merge**:

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
  status: { type: enum, values: [draft, review, published], default: draft }
  ejecutable_en: { type: string, required: true, match: "T*" }

aggregate:
  completed: 'len(filter(descendants, .status == "published"))'

links:
  allowed: [blocks, depends]
```

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
rootline graph [path] [--where 'expr']    # Dependency graph (DOT, Mermaid, --check)
rootline explain <file>                   # Trace field origins, derivations, and errors

# Document lifecycle
rootline init [path] [--force]            # Infer .stem (auto-detects hierarchy)
rootline new <file> [--force] [--dry-run] # Scaffold document from effective schema
rootline fix [file|--all]                 # Auto-repair: add fields, fix enums, propose changes
rootline validate --all --where 'expr'   # Validate only records matching filter
rootline migrate [path]                   # Detect schema changes, rename, split, --to-v2, --from-levels

# Tooling
rootline hooks install|uninstall|status   # Git pre-commit hook management
rootline completion bash|zsh|fish         # Shell completion scripts
rootline serve                            # MCP server (JSON-RPC 2.0 over stdio)
```

All commands support `--output json|table` and `--field` for dot-path extraction:

```bash
rootline describe docs/prd/ --field schema.id.next
# "T004"

rootline query --where 'estado == "Pending"' --field path
# docs/projects/P01/tasks/T005-deploy-grafana.md

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

### MCP Server

`rootline serve` starts a **Model Context Protocol (MCP)** server over stdio, exposing 8 tools via JSON-RPC 2.0. AI assistants query Rootline using the same contracts as the CLI.

Configure in Claude Desktop or any MCP client:

```json
{
  "mcpServers": {
    "rootline": {
      "command": "rootline",
      "args": ["serve"]
    }
  }
}
```

Available tools: `query`, `validate`, `describe`, `tree`, `stats`, `explain`, `fix`, `graph`. See [MCP Server docs](docs/json-rpc.md) for full tool catalog.

---

## Documentation

| Topic | Description |
|-------|-------------|
| [Init](docs/init.md) | Schema inference from existing documents |
| [Validate](docs/validate.md) | Validation rules, batch mode, staged checks |
| [Describe](docs/describe.md) | Describe output, field extraction, source tracking |
| [Query Engine](docs/query.md) | Query contract, operators, result shapes |
| [New](docs/new.md) | Document scaffolding from effective schema |
| [Fix & Proposals](docs/fix.md) | Auto-repair, enum correction, field inference |
| [Explain](docs/explain.md) | Field origin tracing, derivation chain, error diagnosis |
| [Tree](docs/tree.md) | Hierarchical view with completion counts |
| [Stats](docs/stats.md) | Summary counts by type and state |
| [Dependency Graph](docs/graph.md) | Wiki-links, link schema, cycle detection, DOT/Mermaid |
| [Derivation Engine](docs/derivation.md) | Derive and aggregate expressions, builtins, linked fields |
| [Schema Migration](docs/migrate.md) | Breaking change detection, field rename, v2 upgrade |
| [Levels & Match](docs/levels.md) | Hierarchical field scoping with match patterns |
| [MCP Server](docs/json-rpc.md) | Tool catalog, setup, JSON-RPC protocol |
| [Extensibility](docs/extensibility.md) | Extractor architecture, future formats |
| [Visual Identity](docs/identity.md) | Logo, colors, usage guidelines |

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
