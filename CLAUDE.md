# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Rootline is a **file-based database and constraint engine** for structured documentation, written in Go. It treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` files with parent-to-child merge semantics.

**Status**: CLI engine complete — all core commands functional. 14 inference detectors (12 data + 2 governance). Requires Go 1.25+.

## Build & Test Commands

```bash
just check              # gofmt check + golangci-lint + go build
just test               # go test ./... -race  (no coverage measurement — fast cycle)
just fmt                # gofmt -l -w
just validate           # rootline validate --all docs/epics/
just fix-docs           # rootline fix --all docs/epics/
just coverage           # go test ./... -coverprofile; print per-package table + total
just coverage-check     # like coverage, but exits 1 if any package < 85% (per .coverage-floors.toml)
```

Run a single test: `go test ./internal/extract/ -run TestName`

Pre-commit hooks run `gofmt` + `golangci-lint` + `gitleaks` automatically (`.githooks/pre-commit`). The `.pre-commit-config.yaml` provides additional checks. Tests use the standard `testing` package — no external test frameworks.

## Architecture

### Package Layout

- `cmd/rootline/` — CLI entry point. Each subcommand is a separate file (validate.go, query.go, describe.go, init.go, new.go, fix.go, tree.go, stats.go, hooks.go, completion.go, migrate.go, graph.go, explain.go, analyze.go, set.go, repair.go, schema.go). Helpers: table.go, filter.go (output formatting and record filtering). Uses cobra with global flags `--output json|jsonl|csv|table`, `--field` (dot-path extraction supporting both object paths like `summary.total` and array projection like `rows[].path`). `query` supports `--select path,estado,titulo,links` for compact row projections (omits full body by default); field values are resolved from derived fields first (populated via `.stem` `source:` rules like `source: body.h1`), then frontmatter; --select output is `ProjectedQueryResult` with `rows: [map[string]any]` instead of full Records. With `--select`, the `--output` flag accepts `jsonl` (JSON Lines: one JSON object per line) and `csv` (CSV with header row); both require `--select` and emit rows in field order with nil values as empty strings. `fix --all` runs the derive pipeline (derive → enrich → aggregate) before validation so aggregate mismatches are detectable. `analyze` supports `--incremental` to filter inferences covered by existing `.stem`; runs 14 detectors (12 data inference + 2 governance: schema coverage, validation gaps). `repair apply --report <file> [--dry-run]` applies repair-surface proposals (correct_value, add_field, migrate_value, etc.) to document frontmatter only — never touches `.stem` files; rejects schema proposals (extend_enum, add_aggregate, etc.). `schema propose <dir> [--incremental]` reads records and emits versioned JSON schema proposals (version 1, kind `rootline/schema-proposals`) without writing any files; `--incremental` skips proposals already covered by existing `.stem` files. `schema apply --report <proposals.json|analyze.json> [--dry-run]` reads schema proposals or analyze inference reports and applies only schema-surface proposals (create_stem, update_stem) to `.stem` files; update_stem now grows the `.stem` by creating field nodes for newly-observed fields (field_type/enum_values/required_field/constant_field), closing the `analyze --incremental` → `schema apply` loop; auto-detects report kind; rejects wrong kind/version and `requires_agent: true` proposals; runs post-apply validation and returns kind `rootline/schema-apply`. `fix --all` now applies data-only repairs; schema proposals (extend_enum, add_aggregate, remove_stem_field) are skipped and returned in a `schema_suggestions` field for review. `graph --open` renders interactive Mermaid diagram in browser. `init --template <repo>` fetches `.stem` from remote GitHub repos.
- `internal/extract/` — Metadata extraction from files (YAML frontmatter from Markdown, wiki-link extraction from body). Extractor interface + registry pattern. Extracts both `[[wiki-links]]` and markdown links `[text](target)`; each `Link` carries `style` (`wikilink`/`markdown`) and `anchor`. External schemes, images, and pure fragments are skipped.
- `internal/rules/` — `.stem` file loading, walk-up discovery (target → `.git` root), top-down merge (parent → child). Merge is type-driven: maps merge at key level, arrays/scalars replace, null removes. Also contains: validation engine (required, enum, non_empty, exists, requires rules), link schema validation, structural directory rules (require_index, min/max_children), describe output formatting, sequence auto-numbering, validation result types (single + batch), v2 match-based field filtering (`match.go`), drift detection between `.stem` and documents (`drift.go`), stem health diagnostics (11 checks: yaml-valid, scope-match, type-consistency, enum-values, rule-field-exists, field-override, aggregated-required, aggregate-formula-coverage, stem-files-exist, monotonic-violations, unknown-check-keys — called by `validate --all` as pre-phase). Engine rejects v0/v1 stems at parse time. Central resolution API in `resolver.go`: `StemChain`, `EffectiveSchema`, and `Resolve` return the stem chain, merged schema, and field provenance; `ClosestStem`/`RootMostStem` helpers provide explicit closest vs. root-most selection. `ResolveLayered(path, root, monotonic bool)` extends resolution with `LayeredResolution` (Layers + Conflicts); in monotonic mode detects type widening, required loosening, enum extension, severity loosening, and structural loosening as violations. `describe` and `explain` JSON output now include `layers` (ordered `.stem` chain) and `provenance` (field→source map) for consumer observability. `links.styles` selects which link styles are governed (default `[wikilink]`); `links.checks` (`resolve`/`anchors`/`encoding`) enables ADO code-wiki checks via `CheckLinks` (case-sensitive resolution, heading-slug anchors, `%20` encoding); `links.checks.cycles: true` opts `graph --check` into failing on link cycles (default: cycles are informational; `--fail-cycles` overrides). Graph respects styles via `FilterLinksByStyles`.
- `internal/index/` — Directory scanner (respects `.stemignore`), file indexing, scope matching.
- `internal/query/` — Query engine with declarative operators: `eq`, `ne`, `in`, `contains`, `exists`, `and`. Field shortcut resolution. Uses `expr-lang/expr` for expression evaluation. `field_check.go` provides `CheckFieldNames()` for pre-flight unknown-field detection with fuzzy suggestions.
- `internal/derive/` — Derivation engine using `expr-lang/expr`. Per-record derived fields, hierarchical aggregation (bottom-up from children to index files), builtin functions (slugify, lower, upper, trim, strlen, concat). `EnrichBuiltins` resolves the effective stem per-record via `ResolveForRecord`, so `source:` extracted fields (like `source: body.h1`) respect `match:` scopes — they apply only to matching records, not to the entire directory merge.
- `internal/graph/` — Dependency graph from `[[wiki-links]]` in document bodies. Cycle detection, broken link analysis with fuzzy suggestions (up to 3 similar nodes), target resolution with basename fallback. DOT, Mermaid, and HTML output. `--open` renders interactive Mermaid diagram in browser via embedded HTML template (`html.go`, `templates/graph.html`).
- `internal/infer/` — Schema inference from existing documents (14 detectors: 12 data + 2 governance). Analyzes frontmatter to detect field types, enum values, and required fields. `hierarchy.go` detects directory naming patterns (E##, F##, S###, T###) for hierarchical `.stem` generation with per-level field distribution. Body-aware detectors: `body_sections.go` (section patterns), `invariant_extraction.go` (INV\d+ extraction). Semantic extraction: `formal_dependency.go` (wiki-link deps), `traceability_links.go` (Contribuye a/Cubre/Satisface claims). Structural inference: `structural.go` (require_index, min/max_children, naming inconsistency detection). Governance detectors: `schema_coverage.go` (directories without .stem), `validation_gaps.go` (enum without values, untyped fields, sequence incomplete, required understatement). `scaffold.go` creates minimal `.stem` from observed frontmatter. `report.go` defines AnalyzeReport JSON schema (version: 1). `schema_gen.go` exports reusable schema generation services: `GenerateFlatSchema(ctx, dir, records, opts)` and `GenerateHierarchicalSchema(ctx, dir, records, opts)` return `*rules.StemFile` / `map[string]*rules.StemFile` without writing files; `init` command uses these instead of inline logic.
- `internal/migrate/` — Schema migration: diff detection (field added/removed, type changed, enum changed), breaking change classification, bulk field rename with migration log. `migrate --split` converts flat `.stem` to hierarchical per-level files.
- `internal/e2e/` — End-to-end pipeline integration tests.
- `internal/fuzzy/` — Levenshtein-based fuzzy matching for "did you mean?" suggestions. Shared by validation (enum + required field typos), graph (broken link suggestions), query (unknown field warnings), fix, and proposal packages. Adaptive threshold `max(2, len/3)`.
- `internal/fix/` — Fix application: rewrites frontmatter based on proposals. Uses `fuzzy.Match` for enum correction. `repair.go` exports `ApplyRepair(proposals, dryRun, root)` for data-only bulk repair: accepts repair-surface proposals only, never writes `.stem` files, supports dry-run, post-validates modified files. `ApplySchemaProposals(ctx, report, root)` applies schema-surface proposals (extend_enum) from a separated report; counterpart to `ApplyProposals` for the schema path.
- `internal/templates/` — Remote template fetching for `init --template`. Parses `owner/repo[@tag]` refs, clones from GitHub with timeout and `GIT_TERMINAL_PROMPT=0`, validates YAML, copies `.stem` files preserving relative paths.
- `internal/proposal/` — Proposal analysis engine: detects fixable validation errors and generates typed proposals (extend_enum, correct_value, add_field, migrate_value, etc.). `surface.go` defines `ProposalSurface` enum (schema/repair/bootstrap/migration/diagnostic/requires_agent) and `Surface()` classifier so engines can distinguish schema-mutating from document-repair proposals without inspecting command context.

### Core Pipeline

```
Extraction → Parsing → Rule Loading → Validation → Derivation → Aggregation → Query
```

Derivation evaluates per-record expressions from `.stem` `derive:` fields. Aggregation rolls up values from children to parent index files (README.md) using `.stem` `aggregate:` fields. Both use `expr-lang/expr`.

### Key Design Decisions

- CLI commands call the Core Engine directly and emit stable versioned contracts.
- All JSON output carries `"version": 1` for contract stability.
- `.stem` merge behavior is determined by YAML data type, not field names.
- Version is injected via ldflags at build time (`cmd/rootline/root.go`).

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML parsing
- `github.com/expr-lang/expr` — Expression evaluation for derivation and query filters
- `github.com/yuin/goldmark` — Markdown AST parsing (body-aware inference)
- `golang.org/x/text` — Unicode text processing

## Project Documentation

- `docs/research/` — Pre-research for deferred features (plugin architecture)
- `docs/epics/` — Roadmap for features. Completed: derivation engine (E04/F04), dependency graph (E04/F05), fix proposals (E04/F10), schema migration (E07/F01), v1 stem removal (E12), inference detectors (E13/F02). Pending: aggregate consistency engine (E14), repo best practices (E05).
- Documentation is written in a mix of Spanish and English (field names like `estado`, `tipo`, `ejecutable_en` are in Spanish)

## Rootline as Primary Interface

Use `rootline` CLI as the primary tool for querying project data — not manual file reads, Glob, or Explore agents.

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

## Auto-update

`rootline` uses `picokit/autoupdate` to check for and apply new releases automatically. On each run, it fetches the latest release in the background (goroutine + WaitGroup) and applies any staged update on the next invocation. Local builds (`version == "dev"`) skip all network and cache operations. No opt-out env var — controlled only by build-time version injection.

## CI Workflows

CI/CD uses shared reusable workflows from `pablontiv/crossbeam@v1`:
- `go-ci.yml` — build, test (with 85% coverage threshold), tidy, lint, vuln
- `gitleaks.yml` — secret scanning
- `go-release.yml` — auto-tag + goreleaser release
- `codeql.yml` — CodeQL security scanning (Go)
- `scorecard.yml` — OpenSSF Scorecard, inlined locally rather than via crossbeam because the v1 reusable caps workflow permissions at `read-all`, conflicting with the `security-events: write` the job needs. The Scorecard action additionally rejects publish when `security-events: write` is set at the workflow level — write permissions must live only at the job level (top-level stays `contents: read`). Actions are pinned by SHA (Scorecard best practice); if a pin becomes unresolvable, re-pin via `gh api repos/<owner>/<repo>/git/refs/tags/<version> --jq .object.sha`. Runs nightly; also dispatchable via `gh workflow run "OpenSSF Scorecard"`.

`docs-validate` is repo-specific (runs `rootline validate --all docs/epics/`) and stays inline in `ci.yml`.

**Coverage gates**: The 85% threshold in CI (`coverage-threshold: 85` in crossbeam) is mirrored locally via `.coverage-floors.toml` (`default = 85`, uniform across all packages). The local gate runs via `pkcov` from `github.com/pablontiv/picokit` (see [`picokit/docs/coverage-spec.md`](https://github.com/pablontiv/picokit/blob/main/docs/coverage-spec.md)); rootline cumple coverage-spec v1.0. The pre-push hook (`.githooks/pre-push`) runs `just coverage-check` automatically whenever any `.go` file is included in the push — this blocks the push before CI even runs. Do **not** bypass with `git push --no-verify` except in documented emergencies with an explanation in the commit message.

## Module Path

```
github.com/pablontiv/rootline
```
