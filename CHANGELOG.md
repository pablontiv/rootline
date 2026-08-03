# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Releases are automated via CI from conventional commits.

## [Unreleased]

### Added

- Pull request template now has a dedicated **Related issue** section with a `Closes #<N>` field and a checklist item, so issue linkage stops depending on the author remembering the keyword
- `CHANGELOG.md` (this file) — ecosystem documentation baseline
- GitHub Issues enabled on the repository
- `docs/UPGRADE.md` — migration guide for the root marker requirement
- Root marker support: `.stem` files can now declare `root: true` to establish a schema discovery boundary
- Stem-health check for nested root markers (`nested-root-marker` info-level diagnostic)
- Interactive migration prompt: when running on a terminal, rootline prompts to add the root marker to existing projects
- `migrate --split --force` — overwrite existing child `.stem` files (see the matching **BREAKING** entry below)

### Changed

- License changed from PolyForm Noncommercial 1.0.0 to Apache License 2.0 — commercial use is now permitted
- picokit dependency bumped to its Apache-2.0-relicensed release, so distributed binaries no longer embed noncommercially licensed code
- **BREAKING**: Schema discovery no longer uses `.git` directory as a boundary. Projects must now declare a root marker by adding `root: true` to the project's top-level `.stem` file. Existing projects without a root marker will receive a clear error message with the fix.
- **BREAKING**: Commands that govern a project (`validate`, `fix`, `query`, `tree`, `graph`, `describe`, `explain`, `set`, `stats`) now fail instead of silently succeeding when schema discovery cannot find a `.stem` boundary. This prevents false-green validation runs over zero records.
- **BREAKING**: Projects with `.stem` files but no root marker must add `root: true` to their top-level `.stem` — one line — to use governed commands. On a terminal, rootline offers to add it interactively. Do not use `rootline init --force` to migrate: it re-infers and overwrites an existing schema. See `docs/UPGRADE.md`.
- Schema discovery is now stable: resolution depends only on the target path and filesystem `.stem` files, never on the process working directory.
- `set` command no longer requires a `.git` directory to function.
- `rootline init` now emits `root: true` in generated `.stem` files, marking new projects as their own boundaries.
- **BREAKING**: `migrate --split` now refuses to overwrite existing child `.stem` files. It stats every child target before writing anything and fails with `<path> already exists (use --force to overwrite)`, leaving all files untouched. Invocations that previously succeeded by silently clobbering child stems must now pass `--force`. The root `.stem` at the target is the input being rewritten in place and is never guarded. `--dry-run` annotates each target that would be overwritten.

### Fixed

- `migrate --split` emitted child `.stem` files without a version key, so every child parsed as version 0 and the loader rejected it with `stem version 0 is no longer supported`. Child stems now carry `version: 2`.
- `migrate --split` dropped `root: true` from the regenerated root `.stem`, leaving the project with no declared boundary: walk-up escaped into ancestor `.stem` files and `validate --all` failed with `Schema discovery reached the filesystem root without finding a declared boundary`. The marker is now preserved when present (and not invented when absent).
- Graph and tree output is now deterministic. `graph` derived its node and edge arrays from map iteration, which Go randomizes per run, so JSON, DOT and Mermaid emitted a different order on every invocation. Nodes are now sorted lexically by path, edges by `(source, target, line, type)`. `tree` picked its status column by taking whichever enum-typed schema field the map happened to yield first; it now sorts the enum field names and takes the first, at both decision sites. JSON stays at `"version": 1` — same keys, same types, same elements, only the order is now fixed. Multi-enum corpora see a stabilized status column: this repository's `docs/roadmap` reliably shows `estado` instead of alternating with `tipo`.
- False-green validation: `validate --all` on a directory outside any schema now correctly exits with an error instead of succeeding with zero validated records.
- Schema discovery boundary violations: walking up the filesystem no longer accidentally collects `.stem` files from outside the project (e.g., from home directory).
- Working directory dependence: schema resolution is now stable regardless of where commands are invoked from.

## [v0.10.0] - 2026-05

### Added

- `schema propose` and `schema apply` commands for schema proposal workflow
- `repair apply` for data-only bulk repair from proposals report
- `graph --open` renders interactive Mermaid diagram in browser
- 14 inference detectors (12 data + 2 governance: schema coverage, validation gaps)
- `analyze --incremental` to filter inferences covered by existing `.stem`
- `query --select` for compact row projections with `jsonl`/`csv` output
- `rootline explain` for field provenance display
- `init --template <repo>` fetches `.stem` from remote GitHub repos

### Changed

- `apply` deprecated — redirects to `schema apply` and `repair apply`
- Engine rejects v0/v1 stems at parse time

## [v0.9.0] - 2026-03

### Added

- Derivation engine with per-record expressions and hierarchical aggregation
- Dependency graph from `[[wiki-links]]` with cycle detection and DOT/Mermaid/HTML output
- Fix proposals with `rootline fix` and proposal surface classification
- Schema migration: diff detection, breaking change classification, bulk field rename
- `migrate --split` converts flat `.stem` to hierarchical per-level files
