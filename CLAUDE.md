# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Rootline is a **file-based database and constraint engine** for structured documentation, written in Go. It treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` files with parent-to-child merge semantics.

**Status**: CLI engine complete — all core commands functional. 14 inference detectors (12 data + 2 governance). Requires Go 1.26+.

## Build & Test Commands

```bash
just check              # gofmt check + golangci-lint + go build
just test               # go test ./... -race  (no coverage measurement — fast cycle)
just fmt                # gofmt -l -w
just validate           # rootline validate --all docs/roadmap/
just fix-docs           # rootline fix --all docs/roadmap/
just coverage           # go test ./... -coverprofile; print per-package table + total
just coverage-check     # like coverage, but exits 1 if any package < 85% (per .coverage-floors.toml)
```

Run a single test: `go test ./internal/extract/ -run TestName`

Pre-commit hooks run `gofmt` + `golangci-lint` + `gitleaks` automatically (`.githooks/pre-commit`). The `.pre-commit-config.yaml` provides additional checks. Tests use the standard `testing` package — no external test frameworks.

## Architecture

### Package Layout

- `cmd/rootline/` — CLI entry point. Each subcommand is a separate file (validate.go, query.go, describe.go, init.go, new.go, fix.go, tree.go, stats.go, hooks.go, completion.go, migrate.go, graph.go, explain.go, analyze.go, set.go, repair.go, schema.go). Helpers: table.go, filter.go (output formatting and record filtering). Uses cobra with global flags `--output json|jsonl|csv|table`, `--field` (dot-path extraction supporting both object paths like `summary.total` and array projection like `rows[].path`). `query` supports `--select path,estado,titulo,links` for compact row projections (omits full body by default); field values are resolved from derived fields first (populated via `.stem` `source:` rules like `source: body.h1`), then frontmatter; --select output is `ProjectedQueryResult` with `rows: [map[string]any]` instead of full Records. With `--select`, the `--output` flag accepts `jsonl` (JSON Lines: one JSON object per line) and `csv` (CSV with header row); both require `--select` and emit rows in field order with nil values as empty strings. `query` also supports link-traversal predicates: `--has-inbound "<sub-where>"` / `--has-outbound "<sub-where>"` keep records with ≥1 inbound/outbound edge whose linked record matches the sub-where (same grammar as `--where`, empty string = any linked record); `--inbound-type` / `--outbound-type` restrict edges to one link type; `--graph-root <path>` sets the edge-scan universe (default: the query path — never repo root), and the query path must lie inside it. In traversal mode the scan runs from `--graph-root` with the same link prep as `graph` (`FilterLinksByStyles` + `ResolveMarkdownTargets`), record paths in output stay query-path-relative (rebased after traversal filtering), and traversal predicates AND-compose with `--where` before `--sort`/`--limit`/`--count`. `fix --all` runs the derive pipeline (derive → enrich → aggregate) before validation so aggregate mismatches are detectable. `analyze` supports `--incremental` to filter inferences covered by existing `.stem`; runs 14 detectors (12 data inference + 2 governance: schema coverage, validation gaps). `repair apply --report <file> [--dry-run] [--fill-missing]` applies repair-surface proposals (correct_value, add_field, migrate_value, etc.) to document frontmatter only — never touches `.stem` files; rejects schema proposals (extend_enum, add_aggregate, etc.). Each written file is post-validated on its own and restored to its pre-write bytes if validation rejects the result, so a run never leaves a document its own schema refuses; the check is per file, so an unrelated failure elsewhere in the run (including an unreadable path) never disables it. Reverted files appear in a `rolled_back` array (`{path, errors}`) and are withdrawn from `changed`. Both `repair apply` and `schema apply` exit non-zero exactly when `errors[]` or `rolled_back[]` is non-empty; `rejected[]` (policy refusals, including containment violations on either command) and `skipped[]` alone exit 0. `rolled_back[]` is a separate condition, not a subset of `errors[]` — a successful revert leaves `errors[]` empty. The payload is always emitted on stdout before the non-zero exit, and `--dry-run` follows the same rule. Both commands resolve a report's document paths against the SCANNED directory via one shared precedence chain in `cmd/rootline/apply_root.go` (`resolveReportRoot`): `--root` flag > report `root` (absolute) > report `path` > the report file's own directory (the pre-`root` fallback, kept so older reports still apply). `proposal.Report` carries `path`/`root` (populated by `fix --all`), mirroring `SchemaProposalsReport`. The resolved root is echoed in both output envelopes as `root`. Every production write in `internal/fix/` (both `repair.go` and the `fix --all` pipeline in `fix.go`, documents and `.stem` alike) goes through `fix.WriteFileAtomic` — staging file in the target's OWN directory, `Sync`, explicit `Chmod` (because `os.CreateTemp` makes 0600), then rename — so a file is never observed half-written and a failed write leaves the target untouched with no staging debris. The run-level contract is deliberately best-effort, NOT all-or-nothing: a run that fails partway leaves earlier writes in place and reports exactly where it got to. Buffering the whole run was rejected — it would discard the good repairs in a report because of one unreadable path, and would still not survive a kill. Both envelopes carry `complete` (true iff the run carried through everything it accepted — the same condition as exit 0, set by one `seal()` per result type), so a stored artifact is classifiable without re-deriving the rule. Frontmatter rewrites go through a `yaml.Node` round-trip that preserves key order and YAML comments, so a one-field mutation yields a one-field diff (inter-token whitespace and nested indentation are normalized by the encoder). `schema propose <dir> [--incremental]` reads records and emits versioned JSON schema proposals (version 1, kind `rootline/schema-proposals`) without writing any files; `--incremental` skips proposals already covered by existing `.stem` files. `schema apply --report <proposals.json|analyze.json> [--dry-run]` reads schema proposals or analyze inference reports and writes to `.stem` files through two paths: proposal-keyed (only `create_stem` is handled — it scaffolds a new `.stem`) and inference-keyed, where `internal/infer.ApplySchemaInferences` grows the closest `.stem` from inference *types* (`enum_values`, `required_field`, `constant_field`, `field_type`, `untyped_field`, `sequence_incomplete`), creating field nodes when they are missing and extending them otherwise — this closes the `analyze --incremental` → `schema apply` loop; auto-detects report kind; rejects wrong kind/version and `requires_agent: true` proposals; runs post-apply validation and returns kind `rootline/schema-apply`. `fix --all` now applies data-only repairs; schema proposals (extend_enum, add_aggregate, remove_stem_field) are skipped and returned in a `schema_suggestions` field for review. `add_field` proposals carry `value_source` (`schema_default` | `enum_first` | `empty`) recording where the value came from. Only `schema_default` is written by default: a required field whose schema declares no `default:` is REPORTED, not invented, because filling it would destroy the missing-data signal the author asked for. Pass `--fill-missing` (on `fix` and `repair apply`) to write engine-chosen values too. A proposal with no `value_source` at all is treated as `schema_default` so report files from earlier versions keep applying. `init --template <repo>` fetches `.stem` from remote GitHub repos.
- `internal/extract/` — Metadata extraction from files (YAML frontmatter from Markdown, wiki-link extraction from body). Extractor interface + registry pattern. `frontmatter.go` holds `FrontmatterBounds`, the single line-anchored, fence-aware locator for the **leading** `---` block — shared with `internal/rules` so extraction and validation can never disagree on where frontmatter ends. Body thematic breaks (`---`, `***`, `___`) and fenced code blocks (` ``` `, `~~~`) are ordinary content and are never read as delimiters. A file whose first line is `---` still opens a frontmatter block (Jekyll/Hugo convention). Extracts both `[[wiki-links]]` and markdown links `[text](target)`; each `Link` carries `style` (`wikilink`/`markdown`) and `anchor`. External schemes, images, and pure fragments are skipped.
- `internal/rules/` — `.stem` file loading, walk-up discovery, top-down merge (parent → child). Discovery collects `.stem` files upward from the target and stops at the first one carrying `root: true` (the declared governance boundary) or, failing that, at the filesystem root — Git is never consulted. A chain that reaches the filesystem root without a `root: true` marker has no declared boundary, and the boundary preflight (`cmd/rootline/preflight.go`) blocks governed commands: on a terminal it offers to add the marker to the proposed project root, and without one it fails with an error. Commands that create schemas (`init`, `schema`) or resolve none (`completion`, `hooks`, `help`, `migrate`) are exempt. Merge is type-driven: maps merge at key level, arrays/scalars replace, null removes. Also contains: validation engine (required, enum, non_empty, exists, requires rules), link schema validation, structural directory rules (require_index, min/max_children), describe output formatting, sequence auto-numbering, validation result types (single + batch), structural integrity via `ValidateStructure` (scoped to the leading frontmatter block through `extract.FrontmatterBounds`; `multiple_yaml_documents` fires only when that block holds more than one YAML document — an unterminated block is reported by the extractor as `malformed_yaml`), v2 match-based field filtering (`match.go`), drift detection between `.stem` and documents (`drift.go`), stem health diagnostics (12 checks: yaml-valid, scope-match, type-consistency, enum-values, rule-field-exists, field-override, aggregated-required, aggregate-formula-coverage, stem-files-exist, monotonic-violations, unknown-check-keys, nested-root-marker — called by `validate --all` as pre-phase). Engine rejects v0/v1 stems at parse time. **Body-sourced field validation**: Phase 1 validation now resolves fields with `source:` directives directly, making `required` and `enum` constraints apply to body-extracted values. Resolution order: frontmatter value (if present) takes precedence, falling back to body extraction via `extract.ResolveBodyValue` when the field has an `Extract` directive. This works independently of the derive pipeline; validation never depends on prior enrichment. Central resolution API in `resolver.go`: `StemChain`, `EffectiveSchema`, and `Resolve` return the stem chain, merged schema, and field provenance; `ClosestStem`/`RootMostStem` helpers provide explicit closest vs. root-most selection. `ResolveLayered(path, root, monotonic bool)` extends resolution with `LayeredResolution` (Layers + Conflicts); in monotonic mode detects type widening, required loosening, enum extension, severity loosening, and structural loosening as violations. `describe` and `explain` JSON output now include `layers` (ordered `.stem` chain) and `provenance` (field→source map) for consumer observability. `links.styles` selects which link styles are governed (default `[wikilink]`); `links.checks` (`resolve`/`anchors`/`encoding`) enables ADO code-wiki checks via `CheckLinks` (case-sensitive resolution, heading-slug anchors, `%20` encoding); `links.checks.cycles: true` opts `graph --check` into failing on link cycles (default: cycles are informational; `--fail-cycles` overrides). Graph respects styles via `FilterLinksByStyles`.
- `internal/index/` — Directory scanner (respects `.stemignore`), file indexing, scope matching.
- `internal/query/` — Query engine with declarative operators: `eq`, `ne`, `in`, `contains`, `exists`, `and`. Field shortcut resolution. Uses `expr-lang/expr` for expression evaluation. `field_check.go` provides `CheckFieldNames()` for pre-flight unknown-field detection with fuzzy suggestions.
- `internal/derive/` — Derivation engine using `expr-lang/expr`. Per-record derived fields, hierarchical aggregation (bottom-up from children to index files), builtin functions (slugify, lower, upper, trim, strlen, concat). `EnrichBuiltins` resolves the effective stem per-record via `ResolveForRecord`, so `source:` extracted fields (like `source: body.h1`) respect `match:` scopes — they apply only to matching records, not to the entire directory merge.
- `internal/graph/` — Dependency graph from `[[wiki-links]]` in document bodies. Cycle detection, broken link analysis with fuzzy suggestions (up to 3 similar nodes), target resolution with basename fallback. DOT and Mermaid text output (Mermaid renders natively on GitHub and in most editors).
- `internal/infer/` — Schema inference from existing documents (14 detectors: 12 data + 2 governance). Analyzes frontmatter to detect field types, enum values, and required fields. `hierarchy.go` detects directory naming patterns (E##, F##, S###, T###) for hierarchical `.stem` generation with per-level field distribution. Body-aware detectors: `body_sections.go` (section patterns), `invariant_extraction.go` (INV\d+ extraction). Semantic extraction: `formal_dependency.go` (wiki-link deps), `traceability_links.go` (Contribuye a/Cubre/Satisface claims). Structural inference: `structural.go` (require_index, min/max_children, naming inconsistency detection). Governance detectors: `schema_coverage.go` (directories without .stem), `validation_gaps.go` (enum without values, untyped fields, sequence incomplete, required understatement). `scaffold.go` creates minimal `.stem` from observed frontmatter. `report.go` defines AnalyzeReport JSON schema (version: 1). `schema_gen.go` exports reusable schema generation services: `GenerateFlatSchema(ctx, dir, records, opts)` and `GenerateHierarchicalSchema(ctx, dir, records, opts)` return `*rules.StemFile` / `map[string]*rules.StemFile` without writing files; `init` command uses these instead of inline logic.
- `internal/migrate/` — Schema migration: diff detection (field added/removed, type changed, enum changed), breaking change classification, bulk field rename with migration log. `migrate --split` converts flat `.stem` to hierarchical per-level files.
- `internal/e2e/` — End-to-end pipeline integration tests.
- `internal/fix/` — Fix application: rewrites frontmatter based on proposals. Uses `fuzzy.Match` for enum correction. `repair.go` exports `ApplyRepair(proposals, dryRun, root)` for data-only bulk repair: accepts repair-surface proposals only, never writes `.stem` files, supports dry-run, post-validates modified files. `ApplySchemaProposals(ctx, report, root)` applies schema-surface proposals (extend_enum) from a separated report; counterpart to `ApplyProposals` for the schema path.
- `internal/templates/` — Remote template fetching for `init --template`. Parses `owner/repo[@tag]` refs, clones from GitHub with timeout and `GIT_TERMINAL_PROMPT=0`, validates YAML, copies `.stem` files preserving relative paths.
- `internal/gitenv/` — `ClearedEnv()` returns `os.Environ()` minus the repo-scoping git variables (`GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY`, and the rest of that family), preserving PATH/HOME and transport config like `GIT_SSH_COMMAND`. Every `git` invocation that must target its own directory rather than the caller's repository sets `cmd.Env` from it — `internal/templates` (clone plus its test fixtures) and `internal/migrate` (`git show HEAD:<path>`). Without it an inherited `GIT_DIR` redirects the nested command into the caller's repository, which once landed stray commits in this repo. `cmd/rootline/hooks.go` and `cmd/rootline/validate.go` deliberately do NOT use it: they are meant to act on the caller's repository.
- `internal/proposal/` — Proposal analysis engine: detects fixable validation errors and generates typed proposals (extend_enum, correct_value, add_field, migrate_value, etc.). `surface.go` defines `ProposalSurface` enum (schema/repair/bootstrap/migration/diagnostic/requires_agent) and `Surface()` classifier so engines can distinguish schema-mutating from document-repair proposals without inspecting command context.

### Core Pipeline

```
Extraction → Parsing → Rule Loading → Validation → Derivation → Aggregation → Query
```

Derivation evaluates per-record expressions from `.stem` `derive:` fields. Aggregation rolls up values from children to parent index files (README.md) using `.stem` `aggregate:` fields. Both use `expr-lang/expr`.

### Key Design Decisions

- CLI commands call the Core Engine directly and emit stable versioned contracts.
- Each JSON payload carries its own `version` for contract stability. Most commands use version 1; `tree` uses version 2.
- `.stem` merge behavior is determined by YAML data type, not field names.
- Version is injected via ldflags at build time (`cmd/rootline/root.go`).

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML parsing
- `github.com/expr-lang/expr` — Expression evaluation for derivation and query filters
- `github.com/yuin/goldmark` — Markdown AST parsing (body-aware inference)
- `github.com/pablontiv/picokit/fuzzy` — Levenshtein-based fuzzy matching for "did you mean?" suggestions (adaptive threshold `max(2, len/3)`). Shared by validation (enum + required field typos), graph (broken link suggestions), query (unknown field warnings), stem health (unknown check keys), fix, and proposal packages.
- `golang.org/x/text` — Unicode text processing

## Project Documentation

- `docs/research/` — Pre-research for deferred features (plugin architecture)
- `docs/roadmap/` — Feature roadmap as Rootline-governed records (`O##` objectives with `T###` tasks, `.stem`-validated). Query it with the CLI (`rootline query docs/roadmap/ --where "estado == 'Pending'"`) instead of reading files manually.
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

### Version Strategy

Versions are derived from conventional commits by the release workflow, with MAJOR releases reserved for an explicit maintainer decision:

| Commit type | Bump |
|---|---|
| `fix`, `perf` | patch |
| `feat` | minor |
| `feat!`, `fix!` (breaking `!`) | minor |

`docs`, `test`, `refactor`, `ci`, `chore`, and `style` remain non-release-worthy. If the complete commit range since the latest tag contains only those types, the workflow produces no tag and no GitHub Release. This removes the previous contradiction where this table documented those types as `none` while CI still created a PATCH release for them.

## Release Flow

Release-worthy pushes to `master` are automated via CI. The `go-release` reusable workflow (from `pablontiv/crossbeam@v1`) analyzes conventional commits since the latest tag, creates a PATCH or MINOR tag, and triggers goreleaser to build multi-platform binaries. A range containing only `docs`, `test`, `refactor`, `ci`, `chore`, or `style` commits stops without creating a tag or release. Smoke tests verify `--version` and `--help` before creating GitHub Releases. Version is injected at build time via `-ldflags -X main.version={{.Version}}`.

MAJOR releases are deliberate. Before cutting one, a maintainer must review the full commit range since the latest tag, confirm that the compatibility break warrants a MAJOR, and verify that `master` is green and points at the exact commit to release. Then run `gh workflow run ci.yml --ref master -f force-bump=major`, or dispatch the **CI** workflow from the Actions UI with `master` selected and **Force a deliberate major release** set to `major`. The dispatch runs the normal quality gates and passes `force-bump: major` to crossbeam; do not use it to compensate for malformed commit history or a failing branch.

The Justfile contains only development recipes (`check`, `test`, `fmt`, `validate`). Release logic lives in crossbeam shared workflows.

## Auto-update

`rootline` uses `picokit/autoupdate` to check for and apply new releases automatically within the running version's compatibility boundary. Stable releases update only within the same major; pre-1.0 releases update only within the same minor. A cross-boundary release is retained rather than applied, and Rootline prints a short stderr notice with the current and available versions plus deliberate reinstall guidance. On each run, Rootline fetches the latest release in the background (goroutine + WaitGroup) and applies any eligible staged update on the next invocation. Local builds (`version == "dev"`) skip all network and cache operations. There is no opt-out environment variable; the version policy is configured by Rootline.

## CI Workflows

CI/CD uses shared reusable workflows from `pablontiv/crossbeam@v1`:
- `go-ci.yml` — build, test (with 85% coverage threshold), tidy, lint, vuln
- `gitleaks.yml` — secret scanning
- `go-release.yml` — auto-tag + goreleaser release
- `codeql.yml` — CodeQL security scanning (Go)
- `scorecard.yml` — OpenSSF Scorecard, inlined locally rather than via crossbeam because the v1 reusable caps workflow permissions at `read-all`, conflicting with the `security-events: write` the job needs. The Scorecard action additionally rejects publish when `security-events: write` is set at the workflow level — write permissions must live only at the job level (top-level stays `contents: read`). Actions are pinned by SHA (Scorecard best practice); if a pin becomes unresolvable, re-pin via `gh api repos/<owner>/<repo>/git/refs/tags/<version> --jq .object.sha`. Runs nightly; also dispatchable via `gh workflow run "OpenSSF Scorecard"`.

`docs-validate` is repo-specific (runs `rootline validate --all docs/roadmap/`) and stays inline in `ci.yml`.

`installer-smoke` is a post-release job (`needs: [release]`, push only) that runs the public install scripts on native runners — `install.sh` on `ubuntu-latest`/`macos-latest`, `install.ps1` on `windows-latest` — against the freshly published release, asserting `rootline --version`. It exists because the binary-only release smoke never exercised the install path, so a GNU-only `sha256sum --check` in `install.sh` shipped broken on macOS. Native runners (not Docker/qemu) are what make the macOS BSD `sha256sum` and real-Windows `$env:TEMP` paths testable. Inline for now; migrating it to a reusable crossbeam workflow is tracked in `docs/roadmap/T012`.

**Coverage gates**: The 85% threshold in CI (`coverage-threshold: 85` in crossbeam) is mirrored locally via `.coverage-floors.toml` (`default = 85`, uniform across all packages). `pkcov` parses a minimal TOML subset: string arrays in `.coverage-floors.toml` (`packages`, `exclude`) must be single-line. The local gate runs via `pkcov` from `github.com/pablontiv/picokit` (see [`picokit/docs/coverage-spec.md`](https://github.com/pablontiv/picokit/blob/main/docs/coverage-spec.md)); rootline complies with picokit's coverage spec. The pre-push hook (`.githooks/pre-push`) runs `just coverage-check` automatically whenever any `.go` file is included in the push — this blocks the push before CI even runs. Do **not** bypass with `git push --no-verify` except in documented emergencies with an explanation in the commit message.

## Body-Sourced Field Validation

Schema fields with `source:` directives (e.g., `source: body.h1` or `source: body.section["## Heading"]`) now participate in Phase 1 validation. The `required` and `enum` constraints apply to extracted body values, with frontmatter taking precedence:

- **Extraction directives**: `body.h1` extracts the text of the first H1 heading; `body.section["## Heading"]` extracts content under the named heading (e.g., `body.section["## Notes"]`).
- **Frontmatter precedence**: If a field exists in record frontmatter, its value is used directly. Body extraction only occurs when the frontmatter key is absent.
- **Resolution independence**: Validation resolves body-sourced fields directly without depending on the derive pipeline. Both `rootline validate <file>` and `rootline validate --all` use the same resolution logic.
- **Constraint application**: `required: true` fields fail if extraction yields empty or no match; `enum: [values...]` fields validate the extracted value against the allowed list.
- **Match scoping**: Fields with `match:` blocks are filtered by `ResolveForRecord` upstream, so non-matching records never have the field in their effective schema.
- **Breaking**: extending `enum` to extracted values can fail a document that passed before. A body-sourced field used to resolve to nothing, so its `values:` list never ran; now the extracted prose is checked against it. A `.stem` pairing `values:` with `source: body.section[...]` over free-form text will start reporting `enum` errors. Drop `values:`, widen the list, or set `severity: off`.

**Implementation**: `internal/extract/body.go` exports `ResolveBodyValue(record, directive)` for unified body extraction. `internal/rules/validate.go` calls `resolveFieldValue()` during Phase 1 auto-checks to apply frontmatter-first resolution to both `required` and `enum` checks.

## Module Path

```
github.com/pablontiv/rootline
```
