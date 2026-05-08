---
name: rootline
description: Use when working with Markdown records governed by .stem schemas or when the user asks to validate, fix, query, inspect, scaffold, mutate, analyze, apply, graph, trace, or serve Rootline data, even if they do not name Rootline. Do not use for roadmap decomposition or Go debugging.
---

# Rootline CLI Operations

Rootline is the primary interface for `.stem`-governed Markdown data. Use `rootline` commands before manual file reads.

## Source of Truth

Before relying on a command detail, verify it locally with one of:

```bash
rootline <command> --help
rg -n "Use:|Flags\(\)" cmd/rootline/<command>.go internal/mcp/tools.go
```

Use `Read` only for body content the CLI does not expose, a small error context, or editing review.

## Deterministic Execution Rules

1. **Resolve target**: if a command reads or mutates an existing path, verify the path exists. If absent, stop and report the missing path; do not choose a substitute.
2. **Parse with JSON only when supported**: use `--output json` for `validate`, `describe`, `query`, `tree`, `stats`, `explain`, `analyze`, `migrate` diff/rename outputs, and `fix --all --dry-run`. Do not assume JSON for `new`, `set`, single-file `fix`, `graph --check`, or `init`.
3. **Validation exit code**: `validate` returns non-zero when errors exist. If JSON was requested, parse stdout and continue the workflow.
4. **Mutations**: run the command-specific preview first. Apply changes only when the user explicitly requested a write (`arregla`, `aplica`, `cambia`, `set`, `crea`, `corrige`, `fix`, `apply`) or approves the preview.
5. **Verify writes**: after any mutation, run the smallest matching `rootline validate` command and show `git diff -- <target>` when files changed.
6. **Do not use `apply --dry-run` as a safe preview for schema changes**: `apply` can write `.stem` files while `--dry-run` is set. Treat `apply` as mutating unless code inspection of this repository proves otherwise.
7. **Expressions**: `--where` uses expr syntax: `==`, `!=`, `in`, `contains`, `&&`, `||`, booleans, and `field != nil` for existence.
8. **Field extraction**: `--field a.b.c` extracts one JSON dot path. Do not rely on multiple `--field` values.

## Command Routing

| Intent | Command | Canonical form |
|---|---|---|
| Validate files | `validate` | file: `rootline validate file.md -o json`; dir/all: `rootline validate --all <dir> -o json` |
| Repair validation issues | `fix` | dir/all: `rootline fix --all <dir> --dry-run -o json`; file: `rootline fix file.md --dry-run` |
| Inspect schema | `describe` | `rootline describe <path> -o json` |
| Create document | `new` | `rootline new <file.md> --dry-run` then `rootline new <file.md>` |
| Set fields/sections | `set` | `rootline set --dry-run file.md field=value` then apply without `--dry-run` |
| Search records | `query` | `rootline query <dir> --where "estado == 'Pending'" -o json` |
| Show hierarchy | `tree` | `rootline tree <dir> --where "isIndex == false" -o table` |
| Count records | `stats` | `rootline stats <dir> --where "tipo == 'task'" -o json` |
| Explain field origins | `explain` | `rootline explain file.md -o json` |
| Follow reference chains | `trace` | `rootline trace file.md --format json` |
| Graph wiki-links | `graph` | JSON: `rootline graph <dir> -o json`; Mermaid: `rootline graph <dir> -o table --format mermaid` |
| Infer schema | `init` | `rootline init <dir> --dry-run` then `rootline init <dir>` |
| Analyze patterns | `analyze` | `rootline analyze <dir> -o json` |
| Apply analysis | `apply` | treat as write: inspect report, then `rootline apply report.json` |
| Schema operations | `migrate` | diff: `rootline migrate <path> -o json`; writes require `--rename`, `--split`, or `--scaffold` |
| MCP server | `serve` | HTTP: `rootline serve --addr 127.0.0.1:9200`; stdio: `rootline serve --stdio` |
| Git hook management | `hooks` | `rootline hooks status|install|uninstall` |

## Required Workflows

### Validate and Repair

```bash
rootline validate --all <dir> -o json
rootline fix --all <dir> --dry-run -o json
rootline fix --all <dir>
rootline validate --all <dir> -o json
git diff -- <dir>
```

If `<dir>` is a file, use:

```bash
rootline validate <file.md> -o json
rootline fix <file.md> --dry-run
rootline fix <file.md>
rootline validate <file.md> -o json
git diff -- <file.md>
```

### Inspect Required Fields

```bash
rootline describe <file-or-dir> -o json
```

Report: field, type, required, values or sequence `next`, and source `.stem`.

### Create a Document

`new` requires a file path. If the user gives a directory, derive a filename only from explicit user input or from `schema.id.next` plus a user-provided slug.

```bash
rootline describe <dir> --field schema.id.next
rootline new <dir>/<ID>-<slug>.md --dry-run
rootline new <dir>/<ID>-<slug>.md
rootline validate <dir>/<ID>-<slug>.md -o json
```

### Mutate a Field or Section

```bash
rootline set --dry-run <file.md> <field>=<value>
rootline set <file.md> <field>=<value>
rootline validate <file.md> -o json
git diff -- <file.md>
```

Use `field+=value` for section append and `--create` only when the user wants a missing section created.

### Analyze Existing Documents

```bash
rootline analyze <dir> -o json
rootline analyze <dir> --incremental -o json
```

Do not apply results automatically. For `apply`, inspect the report first and treat the command as a write to `.stem` and documents.

## Reference Files

Read only the relevant file:

- Validation and repair: `ref-validate.md`
- Schema inspection and scaffolding: `ref-schema.md`
- Query, tree, stats, explain, trace: `ref-query.md`
- Graph, migrate, init, analyze, apply, serve, MCP: `ref-advanced.md`

## MCP

The MCP catalog is defined in `internal/mcp/tools.go`: `query`, `validate`, `describe`, `tree`, `stats`, `explain`, `fix`, `graph`, `set`, `trace`, `new`, `health`. CLI commands without MCP tools include `init`, `analyze`, `apply`, and `migrate`.
