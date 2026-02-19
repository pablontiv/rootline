# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Rootline is a **file-based database and constraint engine** for structured documentation, written in Go. It treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` files with parent-to-child merge semantics.

**Status**: Early development — cobra CLI skeleton exists with stub commands; core engine packages are placeholder files only.

## Build & Test Commands

```bash
go build ./cmd/rootline/          # Build the binary
go test ./... -race               # Run all tests with race detector
go test ./internal/extract/ -run TestName  # Run a single test
go vet ./...                      # Static analysis
```

No CI pipeline exists yet. Planned: GitHub Actions with `golangci-lint`.

## Architecture

### Package Layout

- `cmd/rootline/` — CLI entry point. Each subcommand is a separate file (query.go, validate.go, describe.go, explain.go, tree.go, stats.go, serve.go). Uses cobra with global flags `--output json|table` and `--field` (dot-path extraction).
- `internal/extract/` — Metadata extraction from files (YAML frontmatter from Markdown). Extractor interface + registry pattern.
- `internal/rules/` — `.stem` file loading, walk-up discovery (target → `.git` root), top-down merge (parent → child). Merge is type-driven: maps merge at key level, arrays/scalars replace, null removes.
- `internal/index/` — Directory scanner (respects `.gitignore`), file indexing.
- `internal/query/` — Query engine with declarative operators: `eq`, `ne`, `in`, `contains`, `exists`, `and`.
- `internal/mcp/` — MCP server via JSON-RPC 2.0 (planned: `modelcontextprotocol/go-sdk`).

### Core Pipeline

```
Extraction → Parsing → Rule Loading → Validation → [Derivation] → Query
```

Derivation is deferred (pipeline slot reserved, not implemented).

### Key Design Decisions

- CLI and MCP server both call the Core Engine directly — same data contracts, no serialization boundary for CLI.
- All JSON output carries `"version": 1` for contract stability.
- `.stem` merge behavior is determined by YAML data type, not field names.
- Version is injected via ldflags at build time (`cmd/rootline/root.go`).

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML parsing (declared in task specs, not yet in go.mod)
- `github.com/stretchr/testify` — Test assertions (planned)
- `modelcontextprotocol/go-sdk` — MCP server (planned)

## Project Documentation

- `docs/intent/v0-rootline.md` — Authoritative design document (architecture, principles, decisions, deferred items)
- `docs/research/` — Investigation notes (query operators, extractors, describe contract, etc.)
- `docs/epics/E03-rootline/` — Implementation roadmap organized as Epic → Feature → Story → Task
- Documentation is written in a mix of Spanish and English (field names like `estado`, `tipo`, `ejecutable_en` are in Spanish)

## Module Path

```
github.com/pones/rootline
```
