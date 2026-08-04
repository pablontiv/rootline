---
name: rootline
description: Use when working with Markdown records governed by .stem schemas or when the user asks to validate, fix, query, inspect, scaffold, mutate, analyze, apply, or graph Rootline data, even if they do not name Rootline. Do not use for roadmap decomposition or Go debugging.
updated: 2026-07-21
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

## Governance Boundary (root marker)

Schema discovery walks up from the target collecting `.stem` files and stops at a `.stem` that declares `root: true`, or at the filesystem root. `.git` is not a boundary. A project must declare where it starts with a `root: true` marker in its top-level `.stem`.

- Governed commands (`validate`, `fix`, `query`, `tree`, `graph`, `describe`, `explain`, `set`, `stats`) fail on a tree with no `.stem` or an unparseable one. Without a terminal they exit non-zero; the error names the exact one-line fix.
- If a governed command reports "Schema discovery reached the filesystem root without finding a declared boundary", the fix is to add `root: true` to the project's top-level `.stem` (one line). Then retry. Do **not** use `init --force` to migrate an existing project — it re-infers and overwrites the schema.
- `rootline init` writes `root: true` for new projects, so they never hit this.
- Bootstrap commands (`schema propose`, `analyze`) work without any `.stem` — they derive one. `init` and `migrate` also run without a marker.

## Deterministic Execution Rules

1. **Resolve target**: if a command reads or mutates an existing path, verify the path exists. If absent, stop and report the missing path; do not choose a substitute.
2. **Parse with JSON only when supported**: use `--output json` for `validate`, `describe`, `query`, `tree`, `stats`, `explain`, `analyze`, `migrate` diff/rename outputs, and `fix --all --dry-run`. Do not assume JSON for `new`, `set`, single-file `fix`, `graph --check`, or `init`.
3. **Validation exit code**: `validate` returns non-zero when errors exist. If JSON was requested, parse stdout and continue the workflow.
4. **Mutations**: run the command-specific preview first. Apply changes only when the user explicitly requested a write (`arregla`, `aplica`, `cambia`, `set`, `crea`, `corrige`, `fix`, `apply`) or approves the preview.
5. **Verify writes**: after any mutation, run the smallest matching `rootline validate` command and show `git diff -- <target>` when files changed.
6. **Schema vs. data mutations are separate**: Use `schema apply --report <file>` for schema proposals and `repair apply --report <file>` for data-only repairs. There is no generic `apply` command. Always inspect proposals before applying.
7. **Expressions**: `--where` uses expr syntax: `==`, `!=`, `in`, `contains`, `&&`, `||`, booleans, and `field != nil` for existence.
8. **Field extraction**: `--field a.b.c` extracts one JSON dot path. Do not rely on multiple `--field` values.

## Command Routing

| Intent | Command | Canonical form |
|---|---|---|
| Validate files | `validate` | file: `rootline validate file.md -o json`; dir/all: `rootline validate --all <dir> -o json` |
| Repair validation issues | `fix` | dir/all: `rootline fix --all <dir> --dry-run -o json > <dir>/repairs.json` (save inside `<dir>`: `repair apply` resolves record paths relative to the report's directory); file: `rootline fix file.md --dry-run` |
| Inspect schema | `describe` | `rootline describe <path> -o json` |
| Create document | `new` | `rootline new <file.md> --dry-run` then `rootline new <file.md>` |
| Set fields | `set` | `rootline set --dry-run file.md field=value` then apply without `--dry-run` |
| Search records | `query` | `rootline query <dir> --where "estado == 'Pending'" -o json` |
| Show hierarchy | `tree` | `rootline tree <dir> --where "isIndex == false" -o table` |
| Count records | `stats` | `rootline stats <dir> --where "tipo == 'task'" -o json` |
| Explain field origins | `explain` | `rootline explain file.md -o json` |
| Graph links (wiki + markdown) | `graph` | JSON: `rootline graph <dir> -o json` (default); DOT: `rootline graph <dir> --format dot -o table`; Mermaid: `rootline graph <dir> --format mermaid -o table` |
| Infer schema | `init` | `rootline init <dir> --dry-run` then `rootline init <dir>` |
| Analyze patterns | `analyze` | `rootline analyze <dir> -o json` |
| Apply schema proposals | `schema apply` | `rootline schema apply --report proposals.json --dry-run` then apply without `--dry-run` |
| Apply data repairs | `repair apply` | `rootline repair apply --report repairs.json --dry-run` then apply without `--dry-run` |
| Schema operations | `migrate` | diff: `rootline migrate <path> -o json`; writes require `--rename`, `--split`, or `--scaffold` |
| Git hook management | `hooks` | `rootline hooks status|install|uninstall` |

## Required Workflows

### Validate and Repair (Data-Only Fixes)

**Use `repair apply` for data-only fixes; use `schema apply` for schema mutations.**

```bash
rootline validate --all <dir> -o json
rootline fix --all <dir> --dry-run -o json > <dir>/repairs.json
rootline repair apply --report <dir>/repairs.json --dry-run -o json
rootline repair apply --report <dir>/repairs.json
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

For schema mutations (extending enums, creating `.stem` fields):

```bash
rootline analyze <dir> -o json  # or rootline schema propose <dir> -o json
rootline schema apply --report <proposals.json> --dry-run -o json
rootline schema apply --report <proposals.json>
rootline validate --all <dir> -o json
```

### Validate and Query Markdown Links (multi-style)

Link extraction always parses both `[[wiki-links]]` and markdown links `[text](target)`. Each link carries `style` (`wikilink`/`markdown`) and optional `anchor`. The `.stem` governs which link styles participate in validation, graph, and query link-traversal operations:

```yaml
version: 2
links:
  styles: [wikilink, markdown]  # governed styles; default [wikilink]
  checks:
    resolve: false              # OPT-OUT only — broken-target detection is on by default
    anchors: true               # #anchor matches a heading slug in the target
    encoding: true              # no raw spaces in targets (use %20)
    cycles: true                # graph --check fails on link cycles (default: informational)
  basename_fallback: false      # opt-in: bare target matches a uniquely-named record anywhere
```

**Validation**: Check failures surface in `validate` as rules `link_resolve` (with fuzzy suggestion), `link_anchor`, `link_encoding`. External schemes, images, and pure fragments are not checked.

**One record set**: `validate` (both modes), `graph`, `query` and `fix --all` apply the same `scope.match` and `.stemignore` filters. `validate <file>` on an excluded file reports a `skipped` warning instead of validating it, so the pre-commit hook and CI agree.

**`resolve` is always on**: broken-target detection needs no declaration (it matches `graph --check`, which never had an opt-in). Set `links.checks.resolve: false` to opt out. `anchors` and `encoding` remain opt-in.

**Basename fallback**: `links.basename_fallback` (default off) lets a path-less target match a uniquely-named record anywhere — the wiki convention. It needs a full-tree index, which `graph` and `query` traversal have and `validate` does not, so with it ON `validate` reports `link_unverifiable` (warning) instead of guessing or staying silent. Off is what makes every command agree.

**`graph --check` runs the declared checks too**: with `links.checks` set it reports `link_anchor` and `link_encoding` alongside cycles and broken links, matching `validate`. Resolution failures appear once, as broken links.

**One resolver**: `validate`, `graph` and `query` traversal share one resolver, so they agree on which links are broken. Wikilinks infer `.md` (`[[b]]`→`b.md`, `[[sub/README]]`→`sub/README.md`), markdown targets resolve literally, `/x.md` resolves against the scan root, and a path-less target matches a uniquely-named record anywhere. A resolved target is never broken even when `scope.match`/`.stemignore` excludes it: the schema declares what is governed, not what exists.

**Query link traversal**: `--has-inbound` and `--has-outbound` predicates search both wiki-link and markdown-link styles (governed by `.stem links.styles`). Combine with `--inbound-type` / `--outbound-type` to restrict to one link type.

**Graph**: `graph` respects the governed link styles from `.stem`. For `graph --check`: `--fail-cycles` overrides the `.stem` cycle opt-in per run (both directions), and `--quiet-cycles` collapses informational cycles to a single summary line (ignored when cycles are failing; JSON output unaffected).

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
