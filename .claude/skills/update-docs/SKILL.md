---
name: update-docs
description: |
  Detect code drift and update project documentation to match current implementation.
  Compares code state against CLAUDE.md, README.md, and docs/*.md command references.
  Does NOT touch docs/epics/ or docs/research/ (roadmap skill territory).
  This skill should be used when the user says "update docs", "sync documentation",
  "actualizar documentación", "docs are stale", "refresh docs",
  "update CLAUDE.md", "update README", "docs out of date",
  "documentation drift", "sync docs with code".
argument-hint: "[claude|readme|commands|all]"
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Grep
  - Glob
  - AskUserQuestion
---

# /update-docs — Documentation Sync

Detect and fix documentation drift between code and reference docs.

## Scope

Target documents (3 categories):

| Category | Files | Source of truth |
|----------|-------|-----------------|
| `claude` | `CLAUDE.md` | `internal/`, `cmd/rootline/*.go`, `go.mod` |
| `readme` | `README.md` | CLI help output, feature implementations |
| `commands` | `docs/*.md` (top-level only) | `rootline <cmd> --help`, cobra definitions |

**Excluded**: `docs/epics/**`, `docs/research/**` — governed by `/roadmap` skill and `.stem` schemas.

## Procedure

### 1. Parse arguments

- `$ARGUMENTS` = `claude` → only CLAUDE.md
- `$ARGUMENTS` = `readme` → only README.md
- `$ARGUMENTS` = `commands` → only docs/*.md
- `$ARGUMENTS` = `all` or empty → all three categories

### 2. Gather current code state

Run these commands via Bash to build a snapshot:

```bash
# CLI subcommands (non-test files)
ls cmd/rootline/*.go | grep -v _test

# Internal packages
ls internal/

# Dependencies
cat go.mod

# Recent changes for context
git log --oneline -20

# Go files changed since last documentation update
git diff --name-only $(git log --format=%H -1 -- README.md)..HEAD -- '*.go'
```

### 3. Detect drift per category

#### CLAUDE.md

Compare these sections against actual code:

| Section | Check against |
|---------|---------------|
| Package Layout — `cmd/rootline/` | Actual `.go` files in `cmd/rootline/` |
| Package Layout — `internal/` | Actual directories in `internal/` with their descriptions |
| Dependencies | `go.mod` requires |
| Build & Test Commands | `.pre-commit-config.yaml`, CI config |
| MCP tools count | Tool registrations in `internal/mcp/` |

Read each source of truth and compare against the corresponding CLAUDE.md section.

#### README.md

| Section | Check against |
|---------|---------------|
| CLI commands | `rootline --help` and `rootline <cmd> --help` output |
| Feature descriptions | Current implementation in `internal/` |
| Documentation table | Actual files in `docs/*.md` |
| Installation | Build commands, release artifacts |

#### docs/*.md

For each command doc file:
- Run `rootline <cmd> --help` and compare flags/usage against documented content
- Check if any new subcommand `.go` files exist without a corresponding doc
- Check if any doc files reference removed commands

### 4. Present drift report

Show the user a summary:

```
DOCUMENTATION DRIFT REPORT
============================

CLAUDE.md
  ✗ Package Layout: internal/infer/ description outdated
  ✗ Dependencies: new dependency not documented

README.md
  ✓ CLI section: up to date
  ✗ Documentation table: missing link to levels.md

docs/
  ✗ docs/migrate.md: --split flag undocumented
  ✓ All commands have doc files
```

### 5. Apply updates

For each category with drift:
1. Read the current document section
2. Read the relevant source code
3. Generate the corrected section
4. Ask the user for confirmation via AskUserQuestion before applying
5. Apply edits using Edit tool (surgical, minimal changes)

### 6. Validate

Run `rootline validate` on any updated docs that are under `.stem` schemas.

### 7. Summary

Report what was updated and suggest a commit message:

```
Updated: CLAUDE.md (Package Layout, Dependencies)
Skipped: README.md (no drift), docs/*.md (no drift)
Suggest: docs: sync CLAUDE.md with current codebase
```
