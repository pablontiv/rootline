# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Releases are automated via CI from conventional commits.

## [Unreleased]

### Added

- `CHANGELOG.md` (this file) — ecosystem documentation baseline
- GitHub Issues enabled on the repository
- `docs/UPGRADE.md` — migration guide for the root marker requirement
- Root marker support: `.stem` files can now declare `root: true` to establish a schema discovery boundary
- Stem-health check for nested root markers (`nested-root-marker` info-level diagnostic)
- Interactive migration prompt: when running on a terminal, rootline prompts to add the root marker to existing projects

### Changed

- License changed from PolyForm Noncommercial 1.0.0 to Apache License 2.0 — commercial use is now permitted
- picokit dependency bumped to its Apache-2.0-relicensed release, so distributed binaries no longer embed noncommercially licensed code
- **BREAKING**: Schema discovery no longer uses `.git` directory as a boundary. Projects must now declare a root marker by adding `root: true` to the project's top-level `.stem` file. Existing projects without a root marker will receive a clear error message with the fix.
- **BREAKING**: Commands that govern a project (`validate`, `fix`, `query`, `tree`, `graph`, `describe`, `explain`, `set`, `stats`) now fail instead of silently succeeding when schema discovery cannot find a `.stem` boundary. This prevents false-green validation runs over zero records.
- **BREAKING**: Projects with `.stem` files but no root marker must add `root: true` to their top-level `.stem` — one line — to use governed commands. On a terminal, rootline offers to add it interactively. Do not use `rootline init --force` to migrate: it re-infers and overwrites an existing schema. See `docs/UPGRADE.md`.
- Schema discovery is now stable: resolution depends only on the target path and filesystem `.stem` files, never on the process working directory.
- `set` command no longer requires a `.git` directory to function.
- `rootline init` now emits `root: true` in generated `.stem` files, marking new projects as their own boundaries.

### Fixed

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
