# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Releases are automated via CI from conventional commits.

## [Unreleased]

### Added

- `CHANGELOG.md` (this file) — ecosystem documentation baseline
- GitHub Issues enabled on the repository

## [v0.10.0] - 2026-05

### Added

- `schema propose` and `schema apply` commands for schema proposal workflow
- `repair apply` for data-only bulk repair from proposals report
- `graph --open` renders interactive Mermaid diagram in browser
- 16 inference detectors (13 data + 3 governance: domain coverage, schema coverage, validation gaps)
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
