# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Rootline is a **file-based database and constraint engine** for structured documentation, written in Go. It treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` files with parent-to-child merge semantics.

**Status**: Engine and MCP server complete — all CLI commands and 8 MCP tools functional. Requires Go 1.24+.

## Build & Test Commands

```bash
just check              # gofmt check + golangci-lint + go build
just test               # go test ./... -race
just fmt                # gofmt -l -w
just validate           # rootline validate --all docs/epics/
just fix-docs           # rootline fix --all docs/epics/
```

Run a single test: `go test ./internal/extract/ -run TestName`

Pre-commit hooks run `gofmt` + `golangci-lint` + `gitleaks` automatically (`.githooks/pre-commit`). The `.pre-commit-config.yaml` provides additional checks. Tests use the standard `testing` package — no external test frameworks.

## Architecture

### Package Layout

- `cmd/rootline/` — CLI entry point. Each subcommand is a separate file (validate.go, query.go, describe.go, init.go, new.go, fix.go, tree.go, stats.go, hooks.go, completion.go, migrate.go, graph.go, explain.go, serve.go, analyze.go, apply.go). Helpers: table.go, filter.go (output formatting and record filtering). Uses cobra with global flags `--output json|table`, `--field` (dot-path extraction). `fix --all` runs the derive pipeline (derive → enrich → aggregate) before validation so aggregate mismatches are detectable. `analyze` supports `--incremental` to filter inferences covered by existing `.stem`. `apply` reads analyze report, modifies `.stem` files (schema inferences) and document frontmatter (data corrections: migrate_value, correct_value, add_field); supports `--dry-run`.
- `internal/extract/` — Metadata extraction from files (YAML frontmatter from Markdown, wiki-link extraction from body). Extractor interface + registry pattern.
- `internal/rules/` — `.stem` file loading, walk-up discovery (target → `.git` root), top-down merge (parent → child). Merge is type-driven: maps merge at key level, arrays/scalars replace, null removes. Also contains: validation engine (required, enum, non_empty, exists, requires rules), link schema validation, structural directory rules (require_index, min/max_children), describe output formatting, sequence auto-numbering, validation result types (single + batch), v2 match-based field filtering (`match.go`), drift detection between `.stem` and documents (`drift.go`), stem health diagnostics (8 checks: yaml-valid, scope-match, type-consistency, enum-values, rule-field-exists, field-override, aggregated-required, aggregate-formula-coverage — called by `validate --all` as pre-phase). Engine rejects v0/v1 stems at parse time.
- `internal/index/` — Directory scanner (respects `.stemignore`), file indexing, scope matching.
- `internal/query/` — Query engine with declarative operators: `eq`, `ne`, `in`, `contains`, `exists`, `and`. Field shortcut resolution. Uses `expr-lang/expr` for expression evaluation.
- `internal/derive/` — Derivation engine using `expr-lang/expr`. Per-record derived fields, hierarchical aggregation (bottom-up from children to index files), builtin functions (slugify, lower, upper, trim, strlen, concat).
- `internal/graph/` — Dependency graph from `[[wiki-links]]` in document bodies. Cycle detection, broken link analysis, target resolution with basename fallback. DOT and Mermaid output.
- `internal/infer/` — Schema inference from existing documents. Analyzes frontmatter to detect field types, enum values, and required fields. `hierarchy.go` detects directory naming patterns (E##, F##, S###, T###) for hierarchical `.stem` generation with per-level field distribution. Body-aware detectors: `body_sections.go` (section patterns), `invariant_extraction.go` (INV\d+ extraction), `subschema_detection.go` (per-type field groups). Semantic extraction: `formal_dependency.go` (wiki-link deps), `traceability_links.go` (Contribuye a/Cubre/Satisface claims). `report.go` defines AnalyzeReport JSON schema (version: 1).
- `internal/migrate/` — Schema migration: diff detection (field added/removed, type changed, enum changed), breaking change classification, bulk field rename with migration log. `migrate --split` converts flat `.stem` to hierarchical per-level files.
- `internal/e2e/` — End-to-end pipeline integration tests.
- `internal/mcp/` — MCP server via JSON-RPC 2.0 over stdio. Registers 8 tools (query, validate, describe, tree, stats, explain, fix, graph) that call core engine directly.
- `internal/fix/` — Fix application: rewrites frontmatter based on proposals.
- `internal/proposal/` — Proposal analysis engine: detects fixable validation errors and generates typed proposals (extend_enum, correct_value, add_field, migrate_value, etc.).

### Core Pipeline

```
Extraction → Parsing → Rule Loading → Validation → Derivation → Aggregation → Query
```

Derivation evaluates per-record expressions from `.stem` `derive:` fields. Aggregation rolls up values from children to parent index files (README.md) using `.stem` `aggregate:` fields. Both use `expr-lang/expr`.

### Key Design Decisions

- CLI and MCP server both call the Core Engine directly — same data contracts, no serialization boundary for CLI.
- All JSON output carries `"version": 1` for contract stability.
- `.stem` merge behavior is determined by YAML data type, not field names.
- Version is injected via ldflags at build time (`cmd/rootline/root.go`).

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML parsing
- `github.com/expr-lang/expr` — Expression evaluation for derivation and query filters
- `github.com/modelcontextprotocol/go-sdk` — MCP server (JSON-RPC 2.0 over stdio)
- `github.com/yuin/goldmark` — Markdown AST parsing (body-aware inference)
- `golang.org/x/text` — Unicode text processing

## Project Documentation

- `docs/research/` — Pre-research for deferred features (plugin architecture)
- `docs/epics/` — Roadmap for features. Completed: derivation engine (E04/F04), dependency graph (E04/F05), fix proposals (E04/F10), schema migration (E07/F01), MCP server (E03/F05), v1 stem removal (E12), inference detectors (E13/F02). Pending: aggregate consistency engine (E14), marketplace distribution (E03/F09), repo best practices (E05).
- Documentation is written in a mix of Spanish and English (field names like `estado`, `tipo`, `ejecutable_en` are in Spanish)

## Rootline as Primary Interface

Use `rootline` CLI as the primary tool for querying project data — not manual file reads, Glob, or Explore agents. When a skill defines its own discovery procedure (e.g., `/roadmap loop` uses `rootline query`), follow the skill's procedure directly instead of launching Explore agents or reading files individually.

- `rootline query` — find records by frontmatter fields (estado, tipo, etc.)
- `rootline tree` — view directory structure with metadata
- `rootline validate` / `rootline fix` — verify and correct files against `.stem` schemas

All transversal commands (`tree`, `stats`, `graph`, `validate --all`) support `--where "expr"` for filtering records with the same expr-lang syntax as `query`.

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

Format: `type(scope): description` — scope is optional. Add `!` before `:` for breaking changes.

### Pre-1.0 Version Strategy

While in v0.x, semver bumps follow pre-1.0 convention:

| Commit type | Bump | Example |
|---|---|---|
| `fix`, `perf` | patch | v0.9.0 → v0.9.1 |
| `feat` | patch | v0.9.0 → v0.9.1 |
| `feat!`, `fix!` (breaking) | minor | v0.9.0 → v0.10.0 |

After v1.0: `feat` bumps minor, breaking bumps major (standard semver).

## Release Flow

Releases are fully automated via CI. On push to `master`, the `auto-tag` job analyzes conventional commits since the last tag, creates version tags, and triggers goreleaser to build multi-platform binaries. Smoke tests verify `--version` and `--help` before creating GitHub Releases. Version is injected at build time via `-ldflags -X main.version={{.Version}}`.

No manual release steps are needed — just push to master with conventional commit messages. The Justfile contains only development recipes (`check`, `test`, `fmt`, `validate`). Release logic lives exclusively in CI to avoid duplication.

## Module Path

```
github.com/pablontiv/rootline
```
