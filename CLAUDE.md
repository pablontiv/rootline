# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Rootline is a **file-based database and constraint engine** for structured documentation, written in Go. It treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` files with parent-to-child merge semantics.

**Status**: Core engine complete. 26/28 CLI commands implemented. Stubs: `explain` (derivation tracing) and `serve` (MCP server).

## Build & Test Commands

```bash
go build ./cmd/rootline/          # Build the binary
go test ./... -race               # Run all tests with race detector
go test ./internal/extract/ -run TestName  # Run a single test
go vet ./...                      # Static analysis
```

Local linting via `.golangci.yml` + `.pre-commit-config.yaml`. CI pipeline planned (GitHub Actions).

## Architecture

### Package Layout

- `cmd/rootline/` — CLI entry point. Each subcommand is a separate file (validate.go, query.go, describe.go, init.go, new.go, fix.go, tree.go, stats.go, doctor.go, hooks.go, completion.go, table.go, explain.go, serve.go). Uses cobra with global flags `--output json|table` and `--field` (dot-path extraction).
- `internal/extract/` — Metadata extraction from files (YAML frontmatter from Markdown). Extractor interface + registry pattern.
- `internal/rules/` — `.stem` file loading, walk-up discovery (target → `.git` root), top-down merge (parent → child). Merge is type-driven: maps merge at key level, arrays/scalars replace, null removes. Also contains: validation engine (required, enum, non_empty, exists, requires rules), describe output formatting, validation result types (single + batch).
- `internal/index/` — Directory scanner (respects `.stemignore`), file indexing, scope matching.
- `internal/query/` — Query engine with declarative operators: `eq`, `ne`, `in`, `contains`, `exists`, `and`. Field shortcut resolution.
- `internal/infer/` — Schema inference from existing documents. Analyzes frontmatter to detect field types, enum values, and required fields.
- `internal/mcp/` — MCP server via JSON-RPC 2.0 (planned, stub only: `modelcontextprotocol/go-sdk`).

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
- `gopkg.in/yaml.v3` — YAML parsing
- `modelcontextprotocol/go-sdk` — MCP server (planned, stub at `internal/mcp/`)

## Project Documentation

- `docs/research/` — Pre-research for deferred features (I2 plugin architecture, I3 derivation engine, I9 opportunity areas)
- `docs/epics/` — Roadmap for pending features only (MCP server, derivation engine, dependency graph). Implemented features have no planning docs — the code is the documentation.
- Documentation is written in a mix of Spanish and English (field names like `estado`, `tipo`, `ejecutable_en` are in Spanish)

## Module Path

```
github.com/pablontiv/rootline
```
