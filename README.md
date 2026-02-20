# Rootline

[![CI](https://github.com/pablontiv/rootline/actions/workflows/ci.yml/badge.svg)](https://github.com/pablontiv/rootline/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: PolyForm Noncommercial](https://img.shields.io/badge/License-PolyForm%20Noncommercial-blue)](LICENSE)

A **file-based database and constraint engine** for structured documentation.

Rootline treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` files.

> **Status**: Core engine functional — validation, query, describe, and CLI complete.
> Features marked **(planned)** are designed but not yet implemented.

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Idea](#core-idea)
- [The `.stem` File](#the-stem-file)
- [CLI](#cli)
- [AI-Native (Planned)](#ai-native-planned)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

---

## Installation

### From source

```bash
go install github.com/pablontiv/rootline/cmd/rootline@latest
```

### Build locally

```bash
git clone https://github.com/pablontiv/rootline.git
cd rootline
go build ./cmd/rootline/
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

# 4. Query — find documents by metadata
rootline query --where 'status eq published'

# 5. Scaffold — create a new document from the schema
rootline new docs/api/auth.md
```

---

## Core Idea

Documentation already has structure. Rootline makes it **explicit**, **inherited**, and **queryable**.

- The directory tree defines hierarchy
- Rules flow from parent to child via `.stem` files
- State is derived, not manually maintained
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
version: 1

scope:
  match: "*.md"

schema:
  title:
    type: string
    required: true
  status:
    type: enum
    values: [draft, review, published]
    default: draft

validate:
  - field: title
    rule: non_empty
  - rule: requires
    if:
      status: published
    then:
      fields: [owner]
```

---

## CLI

Rootline ships as a **single static Go binary** with no dependencies.

```bash
# Core
rootline validate [file|--all|--staged]
rootline query --where 'field op value'
rootline describe <path>
rootline tree [path]
rootline stats [path]

# Document lifecycle
rootline init [path]          # Infer .stem from existing documents
rootline new <file>           # Scaffold document from effective schema
rootline fix <file|--all>     # Repair files (add missing fields, fix values)

# Tooling
rootline doctor               # Check .stem configuration health
rootline hooks install|uninstall|status
rootline completion bash|zsh|fish|powershell

# Planned
rootline explain <file>       # Trace derivation chain
rootline serve                # MCP server
```

All commands support `--output json|table` and `--field` for dot-path extraction:

```bash
rootline describe docs/prd/ --field schema.id.next
# 302

rootline query --where 'estado eq Pending' --field path
# docs/epics/E01/F01/S001/T005-deploy-grafana.md
```

---

## AI-Native (Planned)

Rootline will embed a **Model Context Protocol (MCP)** server. AI assistants will query Rootline using the same contracts as the CLI, making documentation a structured, explainable knowledge source.

---

## Documentation

| Topic | Description |
|-------|-------------|
| [Query Engine](docs/query.md) | Query contract, operators, result shapes |
| [Describe](docs/describe.md) | Describe output, field extraction, source tracking |
| [JSON-RPC Protocol](docs/json-rpc.md) | MCP server protocol (planned) |
| [Extensibility](docs/extensibility.md) | Extractor architecture, future formats |
| [Visual Identity](docs/identity.md) | Logo, colors, usage guidelines |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

---

## License

[PolyForm Noncommercial 1.0.0](LICENSE) — free for non-commercial use.
