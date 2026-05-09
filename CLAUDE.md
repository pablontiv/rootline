# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Rootline is a **file-based database and constraint engine** for structured documentation, written in Go. It treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` files with parent-to-child merge semantics.

**Status**: Engine and MCP server complete — all CLI commands and 9 MCP tools functional. 16 inference detectors (13 data + 3 governance). Requires Go 1.25+.

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

- `cmd/rootline/` — CLI entry point. Each subcommand is a separate file (validate.go, query.go, describe.go, init.go, new.go, fix.go, tree.go, stats.go, hooks.go, completion.go, migrate.go, graph.go, explain.go, serve.go, analyze.go, apply.go, set.go, repair.go). Helpers: table.go, filter.go (output formatting and record filtering). Uses cobra with global flags `--output json|table`, `--field` (dot-path extraction). `fix --all` runs the derive pipeline (derive → enrich → aggregate) before validation so aggregate mismatches are detectable. `analyze` supports `--incremental` to filter inferences covered by existing `.stem`; runs 16 detectors (13 data inference + 3 governance: domain coverage, schema coverage, validation gaps). `apply` reads analyze report, modifies `.stem` files (schema inferences) and document frontmatter (data corrections: migrate_value, correct_value, add_field); supports `--dry-run` (true no-write: schema updates and `missing_schema` scaffolds are skipped). `apply` has a pre-phase that scaffolds `.stem` files for `missing_schema` inferences before processing schema modifications. `repair apply --report <file> [--dry-run]` applies repair-surface proposals (correct_value, add_field, migrate_value, etc.) to document frontmatter only — never touches `.stem` files; rejects schema proposals (extend_enum, add_aggregate, etc.). `graph --open` renders interactive Mermaid diagram in browser. `init --template <repo>` fetches `.stem` from remote GitHub repos. `describe --by-domain` filters schema output by semantic domain.
- `internal/extract/` — Metadata extraction from files (YAML frontmatter from Markdown, wiki-link extraction from body). Extractor interface + registry pattern.
- `internal/rules/` — `.stem` file loading, walk-up discovery (target → `.git` root), top-down merge (parent → child). Merge is type-driven: maps merge at key level, arrays/scalars replace, null removes. Also contains: validation engine (required, enum, non_empty, exists, requires rules), link schema validation, structural directory rules (require_index, min/max_children), describe output formatting, sequence auto-numbering, validation result types (single + batch), v2 match-based field filtering (`match.go`), drift detection between `.stem` and documents (`drift.go`), domain semantic types (`domains.go` — 12 core domains with base type inference, scope-aware field lookup, virtual alias support), stem health diagnostics (11 checks: yaml-valid, scope-match, type-consistency, enum-values, rule-field-exists, field-override, aggregated-required, aggregate-formula-coverage, domain-type-compat, domain-duplicate-scope, monotonic-violations — called by `validate --all` as pre-phase). Engine rejects v0/v1 stems at parse time. Central resolution API in `resolver.go`: `StemChain`, `EffectiveSchema`, and `Resolve` return the stem chain, merged schema, and field provenance; `ClosestStem`/`RootMostStem` helpers provide explicit closest vs. root-most selection. `apply.go` uses this API instead of ad-hoc `entries[0]` indexing. `ResolveLayered(path, root, monotonic bool)` extends resolution with `LayeredResolution` (Layers + Conflicts); in monotonic mode detects type widening, required loosening, enum extension, severity loosening, and structural loosening as violations. `describe` and `explain` JSON output now include `layers` (ordered `.stem` chain) and `provenance` (field→source map) for consumer observability.
- `internal/index/` — Directory scanner (respects `.stemignore`), file indexing, scope matching.
- `internal/query/` — Query engine with declarative operators: `eq`, `ne`, `in`, `contains`, `exists`, `and`. Field shortcut resolution. Uses `expr-lang/expr` for expression evaluation. `field_check.go` provides `CheckFieldNames()` for pre-flight unknown-field detection with fuzzy suggestions.
- `internal/derive/` — Derivation engine using `expr-lang/expr`. Per-record derived fields, hierarchical aggregation (bottom-up from children to index files), builtin functions (slugify, lower, upper, trim, strlen, concat).
- `internal/graph/` — Dependency graph from `[[wiki-links]]` in document bodies. Cycle detection, broken link analysis with fuzzy suggestions (up to 3 similar nodes), target resolution with basename fallback. DOT, Mermaid, and HTML output. `--open` renders interactive Mermaid diagram in browser via embedded HTML template (`html.go`, `templates/graph.html`).
- `internal/infer/` — Schema inference from existing documents (16 detectors: 13 data + 3 governance). Analyzes frontmatter to detect field types, enum values, and required fields. `hierarchy.go` detects directory naming patterns (E##, F##, S###, T###) for hierarchical `.stem` generation with per-level field distribution. Body-aware detectors: `body_sections.go` (section patterns), `invariant_extraction.go` (INV\d+ extraction), `subschema_detection.go` (per-type field groups). Semantic extraction: `formal_dependency.go` (wiki-link deps), `traceability_links.go` (Contribuye a/Cubre/Satisface claims). Structural inference: `structural.go` (require_index, min/max_children, naming inconsistency detection). Governance detectors: `domain_coverage.go` (fields without domain), `schema_coverage.go` (directories without .stem), `validation_gaps.go` (enum without values, untyped fields, sequence incomplete, required understatement). `scaffold.go` creates minimal `.stem` from observed frontmatter. `report.go` defines AnalyzeReport JSON schema (version: 1). `schema_gen.go` exports reusable schema generation services: `GenerateFlatSchema(ctx, dir, records, opts)` and `GenerateHierarchicalSchema(ctx, dir, records, opts)` return `*rules.StemFile` / `map[string]*rules.StemFile` without writing files; `init` command uses these instead of inline logic.
- `internal/migrate/` — Schema migration: diff detection (field added/removed, type changed, enum changed), breaking change classification, bulk field rename with migration log. `migrate --split` converts flat `.stem` to hierarchical per-level files.
- `internal/e2e/` — End-to-end pipeline integration tests.
- `internal/mcp/` — MCP server via JSON-RPC 2.0 over stdio. Registers 9 tools (query, validate, describe, tree, stats, explain, fix, graph, set) that call core engine directly.
- `internal/fuzzy/` — Levenshtein-based fuzzy matching for "did you mean?" suggestions. Shared by validation (enum + required field typos), graph (broken link suggestions), query (unknown field warnings), fix, and proposal packages. Adaptive threshold `max(2, len/3)`.
- `internal/fix/` — Fix application: rewrites frontmatter based on proposals. Uses `fuzzy.Match` for enum correction. `repair.go` exports `ApplyRepair(proposals, dryRun, root)` for data-only bulk repair: accepts repair-surface proposals only, never writes `.stem` files, supports dry-run, post-validates modified files.
- `internal/templates/` — Remote template fetching for `init --template`. Parses `owner/repo[@tag]` refs, clones from GitHub with timeout and `GIT_TERMINAL_PROMPT=0`, validates YAML, copies `.stem` files preserving relative paths.
- `internal/proposal/` — Proposal analysis engine: detects fixable validation errors and generates typed proposals (extend_enum, correct_value, add_field, migrate_value, etc.). `surface.go` defines `ProposalSurface` enum (schema/repair/bootstrap/migration/diagnostic/requires_agent) and `Surface()` classifier so engines can distinguish schema-mutating from document-repair proposals without inspecting command context.

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

Releases are fully automated via CI. On push to `master`, the `go-release` reusable workflow (from `pablontiv/crossbeam@v1`) analyzes conventional commits since the last tag, creates version tags, and triggers goreleaser to build multi-platform binaries. Smoke tests verify `--version` and `--help` before creating GitHub Releases. Version is injected at build time via `-ldflags -X main.version={{.Version}}`.

No manual release steps are needed — just push to master with conventional commit messages. The Justfile contains only development recipes (`check`, `test`, `fmt`, `validate`). Release logic lives in crossbeam shared workflows.

## CI Workflows

CI/CD uses shared reusable workflows from `pablontiv/crossbeam@v1`:
- `go-ci.yml` — build, test (with 85% coverage threshold), tidy, lint, vuln
- `gitleaks.yml` — secret scanning
- `go-release.yml` — auto-tag + goreleaser release
- `codeql.yml` — CodeQL security scanning (Go)
- `scorecard.yml` — OpenSSF Scorecard

`docs-validate` is repo-specific (runs `rootline validate --all docs/epics/`) and stays inline in `ci.yml`.

## Module Path

```
github.com/pablontiv/rootline
```
