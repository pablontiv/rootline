# Rootline - AI Context & Development Guide

Rootline is a **file-based database and constraint engine** for structured documentation. It treats the filesystem as a database where directories are tables, files are records, and metadata is stored in YAML frontmatter. Structure and rules are inherited through `.stem` files.

## Project Overview

- **Core Technology**: Go 1.24+
- **Main Dependencies**: 
  - `cobra`: CLI framework
  - `yaml.v3`: YAML parsing for frontmatter and `.stem` files
  - `expr-lang/expr`: Expression evaluation for queries and derivations
- **Key Concepts**:
  - **.stem files**: Define schema, validation, derivation, and aggregation rules. They follow a walk-up discovery and top-down merge pattern (parent to child).
  - **Frontmatter**: Markdown files contain YAML frontmatter which serves as the record's data.
  - **Derivation & Aggregation**: Fields can be computed per-record or rolled up from children to parents using `expr` expressions.
  - **AI-Native**: Designed for tool-use by AI assistants with stable JSON output (`"version": 1`).

## Building and Running

```bash
# Build the binary
go build ./cmd/rootline/

# Run tests with race detector
go test ./... -race

# Run specific package tests
go test ./internal/extract/ -v

# Static analysis and linting
go vet ./...
golangci-lint run ./...
```

## Architecture

- `cmd/rootline/`: CLI entry point. Each subcommand (e.g., `validate.go`, `query.go`) is a separate file.
- `internal/extract/`: Handles metadata (YAML) and link extraction from files.
- `internal/rules/`: Manages `.stem` loading, inheritance, and the validation engine.
- `internal/index/`: Directory scanning and file indexing.
- `internal/query/`: The query engine using `expr-lang/expr`.
- `internal/derive/`: Evaluation of derived and aggregated fields.
- `internal/graph/`: Dependency graph construction from wiki-links (`[[link]]`).
- `internal/migrate/`: Schema evolution and migration tools.

## Development Conventions

- **Coding Style**: Standard Go (`gofmt`, `go vet`). Use tabs for Go and 2 spaces for YAML/Markdown.
- **Testing**: Standard `testing` package only. No external frameworks.
- **Commit Messages**: **Conventional Commits** are mandatory.
  - Format: `type(scope): description`
  - Types: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `chore`, `perf`, `style`.
- **JSON Stability**: All CLI JSON output must include `"version": 1` to ensure contract stability for automation and AI tool-use.
- **Rootline CLI as Tooling**: When working on this project, you can use the `rootline` binary itself to explore the codebase's own documentation (found in `docs/`).

## Project-Specific Commands

- `rootline init <path>`: Infer `.stem` rules from existing docs.
- `rootline validate --all`: Check all documents against their rules.
- `rootline query --where 'expr'`: Find documents by metadata.
- `rootline explain <file>`: Trace field origins and derivations.
- `rootline graph <path> --check`: Visualize dependencies and detect cycles.
