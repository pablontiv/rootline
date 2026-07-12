---
name: rootline
description: Use when working with Markdown records governed by .stem schemas or when the user asks to validate, fix, query, inspect, scaffold, mutate, analyze, apply, or graph Rootline data, even if they do not name Rootline. Do not use for roadmap decomposition or Go debugging.
updated: 2026-07-10
---

# Rootline CLI Operations

Rootline is the primary interface for `.stem`-governed Markdown data. Use `rootline` commands before manual file reads. Rootline no longer owns a co-located Pi package; Pi integrations live outside this repository.

## Source of Truth

Before relying on a command detail, verify it locally with one of:

```bash
rootline <command> --help
rg -n "Use:|Flags\(\)" cmd/rootline/<command>.go
```

Use `Read` only for body content the CLI does not expose, a small error context, or editing review.

Local coverage check: `just coverage-check` (requires `.coverage-floors.toml`).

## Deterministic Execution Rules

1. **Resolve target**: if a command reads or mutates an existing path, verify the path exists. If absent, stop and report the missing path; do not choose a substitute.
2. **Parse with JSON only when supported**: use `--output json` for `validate`, `describe`, `query`, `tree`, `stats`, `explain`, `analyze`, `migrate` diff/rename outputs, and `fix --all --dry-run`. Do not assume JSON for `new`, `set`, single-file `fix`, `graph --check`, or `init`.
3. **Validation exit code**: `validate` returns non-zero when errors exist. If JSON was requested, parse stdout and continue the workflow.
4. **Mutations**: run the command-specific preview first. Apply changes only when the user explicitly requested a write (`arregla`, `aplica`, `cambia`, `set`, `crea`, `corrige`, `fix`, `apply`) or approves the preview.
5. **Verify writes**: after any mutation, run the smallest matching `rootline validate` command and show `git diff -- <target>` when files changed.
6. **Schema vs. data mutations are now separate**: Use `schema apply --report <file>` for schema proposals and `repair apply --report <file>` for data-only repairs. The legacy `apply` command is deprecated in favor of these specialized commands. Always inspect proposals before applying.
7. **Expressions**: `--where` uses expr syntax: `==`, `!=`, `in`, `contains`, `&&`, `||`, booleans, and `field != nil` for existence.
8. **Field extraction**: `--field a.b.c` extracts one JSON dot path. Do not rely on multiple `--field` values.

## Command Routing

| Intent | Command | Canonical form |
|---|---|---|
| Validate files | `validate` | file: `rootline validate file.md -o json`; dir/all: `rootline validate --all <dir> -o json` |
| Repair validation issues | `fix` | dir/all: `rootline fix --all <dir> --dry-run -o json`; file: `rootline fix file.md --dry-run` |
| Inspect schema | `describe` | `rootline describe <path> -o json` |
| Create document | `new` | `rootline new <file.md> --dry-run` then `rootline new <file.md>` |
| Set fields | `set` | `rootline set --dry-run file.md field=value` then apply without `--dry-run` |
| Search records | `query` | `rootline query <dir> --where "estado == 'Pending'" -o json` |
| Show hierarchy | `tree` | `rootline tree <dir> --where "isIndex == false" -o table` |
| Count records | `stats` | `rootline stats <dir> --where "tipo == 'task'" -o json` |
| Explain field origins | `explain` | `rootline explain file.md -o json` |
| Graph links (wiki + markdown) | `graph` | JSON: `rootline graph <dir> -o json`; Mermaid: `rootline graph <dir> -o table --format mermaid` |
| Infer schema | `init` | `rootline init <dir> --dry-run` then `rootline init <dir>` |
| Analyze patterns | `analyze` | `rootline analyze <dir> -o json` |
| Apply schema proposals | `schema apply` | `rootline schema apply --report proposals.json --dry-run` then apply without `--dry-run` |
| Apply data repairs | `repair apply` | `rootline repair apply --report repairs.json --dry-run` then apply without `--dry-run` |
| Apply analysis (legacy) | `apply` | deprecated; use `schema apply` or `repair apply` instead |
| Schema operations | `migrate` | diff: `rootline migrate <path> -o json`; writes require `--rename`, `--split`, or `--scaffold` |
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

### Validate Markdown Links (multi-style, v1.12.2+)

Link extraction always parses both `[[wiki-links]]` and markdown links `[text](target)`; each link carries `style` (`wikilink`/`markdown`) and `anchor`. The `.stem` governs which styles participate in validation and graph:

```yaml
version: 2
links:
  styles: [markdown]   # governed styles; default [wikilink] (backcompat)
  checks:
    resolve: true      # target exists (case-sensitive; dir targets need README.md)
    anchors: true      # #anchor matches a heading slug in the target
    encoding: true     # no raw spaces in targets (use %20)
    cycles: true       # graph --check fails on link cycles (default: informational)
```

Check failures surface in `validate` as rules `link_resolve` (with fuzzy suggestion), `link_anchor`, `link_encoding`. Absolute targets (`/...`), external schemes, images, and pure fragments are not checked. Use for repos with relative markdown links (e.g. Azure DevOps code wikis).

For `graph --check`: `--fail-cycles` overrides the `.stem` cycle opt-in per run (both directions), and `--quiet-cycles` collapses informational cycles to a single summary line (ignored when cycles are failing; JSON output unaffected).

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

> **Multi-pattern schemas:** `schema.id.next` retorna el próximo valor del primer patrón alfabético que tiene entries existentes en el directorio. En schemas con múltiples patrones de secuencia (ej: `O*` y `T*`), usar `--field schema.id.next_by_pattern` para obtener el próximo valor de **todos** los patrones simultáneamente: `{"O*": "O14", "T*": "T014"}`.

### Mutate a Field or Section

```bash
rootline set --dry-run <file.md> <field>=<value>
rootline set <file.md> <field>=<value>
rootline validate <file.md> -o json
git diff -- <file.md>
```

Use `--create` only when the user wants a missing field created with a value. `--create` does not create files — use `rootline new` to scaffold new documents. `--no-validate` skips post-mutation validation only; pre-validation of enum constraints always runs. Note: `type: section` and section append (`+=`) are removed; use `source: body.section[...]` + `type: string` in the `.stem` instead.

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
- Query, tree, stats, explain: `ref-query.md`
- Graph, migrate, init, analyze, apply: `ref-advanced.md`
