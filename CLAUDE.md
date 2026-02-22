# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Rootline is a **file-based database and constraint engine** for structured documentation, written in Go. It treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` files with parent-to-child merge semantics.

**Status**: Core engine complete. Stubs: `explain` (derivation tracing) and `serve` (MCP server). Requires Go 1.24+.

## Build & Test Commands

```bash
go build ./cmd/rootline/          # Build the binary
go test ./... -race               # Run all tests with race detector
go test ./internal/extract/ -run TestName  # Run a single test
go vet ./...                      # Static analysis
golangci-lint run ./...           # Full lint (govet, errcheck, staticcheck, unused, ineffassign, gocritic)
```

Pre-commit hooks run `golangci-lint` + `gofmt` automatically (`.pre-commit-config.yaml`). Tests use the standard `testing` package — no external test frameworks.

## Architecture

### Package Layout

- `cmd/rootline/` — CLI entry point. Each subcommand is a separate file (validate.go, query.go, describe.go, init.go, new.go, fix.go, tree.go, stats.go, doctor.go, hooks.go, completion.go, table.go, graph.go, explain.go, serve.go). Uses cobra with global flags `--output json|table` and `--field` (dot-path extraction).
- `internal/extract/` — Metadata extraction from files (YAML frontmatter from Markdown). Extractor interface + registry pattern.
- `internal/rules/` — `.stem` file loading, walk-up discovery (target → `.git` root), top-down merge (parent → child). Merge is type-driven: maps merge at key level, arrays/scalars replace, null removes. Also contains: validation engine (required, enum, non_empty, exists, requires rules), describe output formatting, validation result types (single + batch).
- `internal/index/` — Directory scanner (respects `.stemignore`), file indexing, scope matching.
- `internal/query/` — Query engine with declarative operators: `eq`, `ne`, `in`, `contains`, `exists`, `and`. Field shortcut resolution. Uses `expr-lang/expr` for expression evaluation.
- `internal/graph/` — Dependency graph between documents (based on frontmatter references).
- `internal/infer/` — Schema inference from existing documents. Analyzes frontmatter to detect field types, enum values, and required fields.
- `internal/e2e/` — End-to-end pipeline integration tests.
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

## Rootline as Primary Interface

Use `rootline` CLI as the primary tool for querying project data — not manual file reads, Glob, or Explore agents. When a skill defines its own discovery procedure (e.g., `/roadmap loop` uses `rootline query`), follow the skill's procedure directly instead of launching Explore agents or reading files individually.

- `rootline query` — find records by frontmatter fields (estado, tipo, etc.)
- `rootline tree` — view directory structure with metadata
- `rootline validate` / `rootline fix` — verify and correct files against `.stem` schemas

Only fall back to `Read` when you need raw markdown body content that rootline doesn't expose.

## Commit Convention

Commits follow [Conventional Commits](https://www.conventionalcommits.org/). The `.githooks/commit-msg` hook enforces this format.

| Type | Semver Impact | When to use |
|------|--------------|-------------|
| `feat` | minor | New functionality |
| `fix` | patch | Bug fix |
| `docs` | none | Documentation only |
| `test` | none | Adding or updating tests |
| `refactor` | none | Code restructuring, no behavior change |
| `ci` | none | CI/CD pipeline changes |
| `chore` | none | Maintenance, dependencies |
| `perf` | patch | Performance improvement |
| `style` | none | Formatting, whitespace |

Format: `type(scope): description` — scope is optional. Add `!` before `:` for breaking changes (bumps major).

## Release Flow

```bash
svu current              # Show current version (from latest git tag)
svu next                 # Preview next version based on commits since last tag
git tag "$(svu next)" && git push --tags  # Tag and trigger goreleaser
```

Requires `svu` installed: `go install github.com/caarlos0/svu/v2@latest`

## Module Path

```
github.com/pablontiv/rootline
```
