---
estado: Completed
---
# Validation Engine

`rootline validate` checks documents against the effective schema defined by `.stem` files in the directory tree. It supports single-file, batch, and git-staged modes.

## CLI Usage

```bash
rootline validate docs/api/overview.md           # Single file
rootline validate --all docs/epics/               # All files under path
rootline validate --staged                        # Git staged files only
rootline validate --all --strict                  # Warnings as errors
rootline validate --all --where 'estado != "Completed"'  # Filtered
```

**Schema Discovery**: `validate` discovers schemas by walking up the directory tree, collecting `.stem` files until one declares `root: true` — the governance boundary — or, if none does, until the filesystem root. Reaching the filesystem root without a marker is not a valid configuration: the boundary preflight requires one before any governed command runs, offering to add it on a terminal and failing with an error otherwise. A `.stem` file must exist in the target path or a parent directory; if none exists, `validate` exits with error code 1.

**Prerequisite**: Before running `validate`, initialize the directory with `rootline init` to create a `.stem` schema file.

### Flags

| Flag | Description |
|------|-------------|
| `--all` | Validate all files in scope from current directory |
| `--staged` | Validate only files in git staging area |
| `--strict` | Treat warnings as errors (exit code 1) |
| `--where "expr"` | Filter records in `--all` mode (expr-lang syntax, repeatable) |

## Validation Phases (--all mode)

Batch validation runs four phases in order:

1. **Stem Health** — 12 diagnostics on `.stem` files themselves, reported under
   `stem_health` (never as records). This phase runs before the corpus scan and its
   findings survive a scan failure, so a missing or unparseable `.stem` still produces
   the envelope:
   - `stem-files-exist` — the scanned tree contains at least one `.stem` file
   - `yaml-valid` — valid YAML syntax
   - `scope-match` — scope patterns match at least one file
   - `type-consistency` — field types are consistent across hierarchy
   - `field-declaration` — schema fields use supported canonical types, values, and source declarations
   - `rule-field-exists` — validation rules reference defined fields
   - `field-override` — child field overrides warn about partial override
   - `aggregated-required` — warns when a field is both `required` and aggregated (`required` is auto-skipped on index files, so the combination rarely does what it looks like)
   - `aggregate-formula-coverage` — an aggregate formula references every enum value of the field it aggregates
   - `monotonic-violations` — child constraints do not widen parent constraints. Each of the five categories names itself (`widens type`, `loosens required`, `loosens severity`, `enum extended with disallowed value(s)`, and structural bounds reported under their full path such as `structural.subdirs.min_children`)
   - `unknown-check-keys` — keys under `links.checks` are recognized (fuzzy "did you mean?" on typos)
   - `nested-root-marker` — reports (info) a `.stem` that declares `root: true` below another one that already does, since records under it stop inheriting the ancestor

2. **Document Validation** — Checks each record against its effective schema: required fields, enum values, non_empty, exists, requires rules. When the effective `.stem` declares `links.checks`, link targets are resolved here too — see [Link resolution](#link-resolution).
3. **Structural Validation** — Directory-level rules: `require_index` (must have README.md), `min_children`/`max_children` constraints.
4. **Drift Detection** — Warns when an index file's field value contradicts its children (e.g., parent says "Completed" but children are "Pending").

## Link resolution

`links.checks` resolves targets through the same engine `graph` and `query` use, so the
commands cannot disagree about whether a link is broken.

| Target form | Resolves to |
|---|---|
| `[[b]]` (wikilink, no extension) | `b.md` beside the source |
| `[[sub/README]]` | `sub/README.md` |
| `[b](b.md)` (markdown) | `b.md` literally — markdown targets never infer an extension |
| `[x](/docs/Page.md)` (root-anchored) | `docs/Page.md` under the scan root |
| `[x](guides/)` (directory) | `guides/README.md` |
| `[x](my%20page.md)` | `my page.md` — percent escapes decode |

Two rules are deliberate rather than incidental:

- **`.md` is inferred only for wikilinks, and only on the last path component.** A markdown
  destination carries its extension by convention and Azure DevOps resolves it literally, so
  inferring one would accept links the published wiki rejects. A missing intermediate directory
  is a missing directory, not a file to guess at.
- **Matching is case-sensitive on every component.** APFS is case-insensitive; Azure DevOps and
  git are not, so a link that works locally can 404 once published. Rootline reports the
  mismatch rather than hiding it.

Root-anchored targets (`/x.md`) resolve against the scan root. For `validate --all` that is the
directory you pointed the command at; for a single-file `rootline validate <file>` it is the
governance boundary — the directory of the root-most `.stem`. When no schema governs the file,
a root-anchored target cannot be anchored and stays unresolved rather than guessing.

Resolution is clamped to the root. A target that walks out of the tree does not resolve —
whether it escapes with `..` (root-anchored `/../secrets.md` or relative `../../secrets.md`) or
through a symlink inside the tree that points outside it. Containment compares real paths, so a
symlink that stays inside the root still resolves normally. Link targets are document-controlled
text, and resolution would otherwise report on — and with `anchors` enabled, read — files
outside the tree being governed.

### Naming a file does not bypass its exclusions

`validate <file>` and `--staged` apply the same `scope.match` and `.stemignore` filters
`validate --all` applies. Naming a file explicitly cannot smuggle it back into governance —
otherwise the pre-commit hook and CI enforce different rules on the same file, and a record the
schema declares out of scope blocks the commit while passing CI.

The skip is reported rather than silent, as a warning, so a run that checks nothing says why:

```console
$ rootline validate scope/other.md --field "results[].warnings"
[[{"rule":"skipped","field":"",
   "message":"skipped: out of scope for this .stem (scope.match)","severity":"warn"}]]
```

### Broken-target detection is always on

`links.checks.resolve` needs no opt-in. `graph --check` has always failed on a broken link with
no schema declaration, so leaving `validate` silent unless `links.checks` declared `resolve` meant
the two commands disagreed on the one property both claim to check.

A repository that genuinely wants dangling links opts out explicitly:

```yaml
links:
  checks:
    resolve: false
```

`anchors` and `encoding` stay opt-in — only `resolve` flipped.

### Basename fallback (`links.basename_fallback`)

Off by default. Turning it on lets a target that names no path match a uniquely-named record
anywhere in the tree — the wiki convention where `wiki/sources/witness.md` writes
`[[supports:tool-a.md]]` for `wiki/entities/tool-a.md`:

```yaml
links:
  basename_fallback: true
```

**This is the one place the commands intentionally differ, and it is a real trade.** Matching a
bare name needs an index of every record. `graph` and `query` traversal scan the whole tree and
have one; `validate` checks records one at a time and never does. So with the knob on, `validate`
cannot decide such a link. It does not guess and it does not stay quiet — it reports
`link_unverifiable` as a **warning** naming the command that can decide it:

```console
$ rootline validate wiki/sources/witness.md
link target "tool-a.md" cannot be verified: links.basename_fallback matches against every
record, which this command does not scan (check it with 'rootline graph --check')
```

Leaving the knob off is what makes every command answer identically. Turning it on is a
deliberate choice to buy the wiki ergonomic at the cost of one check `validate` can no longer
make — declared in the schema rather than discovered as a silent disagreement.

Resolution asks the filesystem, not the record set. A target that exists on disk resolves even
when `scope.match` or `.stemignore` excludes it from governance: the schema declares what is
*governed*, not what *exists*.

## Output Envelope

Every `validate` invocation emits one shape — one file, several files, `--all`,
`--staged` with an empty index, and the corpus-scan failure path alike. Nothing about
the envelope varies with the flags you passed, so a consumer never branches on how the
command was called.

```json
{
  "version": 2,
  "kind": "rootline/validate-batch",
  "results": [ ... ],
  "structural": [ ... ],
  "stem_health": [ ... ],
  "drift_warnings": [ ... ],
  "notices": [ ... ],
  "summary": { ... }
}
```

All six keys are always present; empty collections are `[]`, never absent and never
`null`.

| Key | Population |
|-----|------------|
| `results` | Documents. One entry per validated record — never a `.stem` file, never a directory. |
| `structural` | Directories checked against `structural:` rules. Same entry shape as `results`, with a trailing-slash path. |
| `stem_health` | Schema diagnostics about `.stem` files. |
| `drift_warnings` | Parent/child divergence between an index file and its children. |
| `notices` | Run-level diagnostics that belong to no single record. |
| `summary` | Counts, derived from the collections above. |

Each population is disjoint, and each is counted on its own axis. Splitting them changed
where a verdict is *reported*, never whether it counts: an error anywhere still exits 1.

### `results[]`

```json
{
  "version": 1,
  "kind": "rootline/validate",
  "path": "docs/draft.md",
  "valid": false,
  "errors": [
    {
      "rule": "enum",
      "field": "estado",
      "message": "value Peding is not in allowed values: [Pending, Completed] (did you mean \"Pending\"?)",
      "source": "docs/.stem",
      "severity": "error",
      "suggestion": "Pending"
    }
  ],
  "warnings": []
}
```

Each error carries the rule, field, message, source `.stem`, severity, and an optional
`suggestion` (fuzzy "did you mean?" hint).

### `structural[]`

```json
{ "version": 1, "kind": "rootline/validate", "path": "sub/",
  "valid": false,
  "errors": [ { "rule": "max_children", "field": "directory", "message": "..." } ],
  "warnings": [] }
```

A directory is not a record, so it no longer occupies a `results` slot. The root of the
scan is reported as `"/"`.

### `stem_health[]`

A `.stem` file is not a record, so a schema defect never occupies a `results` slot and
never reaches `summary.total`. Before this separation, `validate --all` reported 5 on a
three-document corpus with two health findings, while `query --count`, `tree --field
root.total` and `stats --field total` all reported 3 on the same path.

```json
{
  "path": "docs/sub/.stem",
  "check": "scope-match",
  "field": "",
  "severity": "warn",
  "message": "scope.match \"*.txt\" matches no files in directory"
}
```

`severity` is `error`, `warn` or `info`. `info` is a real level, not a demoted warning:
`nested-root-marker` describes a supported configuration and must not fail `--strict`.

Health runs before the corpus scan and survives it. A tree with no `.stem` anywhere, or
one whose `.stem` does not parse, still emits the envelope — carrying `stem-files-exist`
or `yaml-valid` — instead of a raw Go error on stderr and no JSON at all.

### `notices[]`

```json
{ "severity": "error", "code": "scan_failed", "message": "scanning: ..." }
```

| Code | Severity | Meaning |
|------|----------|---------|
| `scan_failed` | error | The corpus could not be scanned. `stem_health` explains why. |
| `schema_resolution_failed` | error | A `.stem` chain failed to resolve during structural or drift checks. |
| `stem_health_unavailable` | warn | Stem health itself could not run. |
| `no_records` | warn | `--all` scanned the path and found no records. |

`no_records` exists because an emptied or renamed path used to report `total: 1,
valid: 1` — the `stem-files-exist` pseudo-record — and a CI gate read that as green.

Switch on `code`; it is stable. `message` is for humans.

### `summary`

```json
{
  "total": 42,
  "valid": 40,
  "invalid": 2,
  "errors_count": 3,
  "warnings_count": 0,
  "drift_warnings_count": 1,
  "structural_errors_count": 0,
  "structural_warnings_count": 0,
  "stem_health_errors_count": 0,
  "stem_health_warnings_count": 2,
  "stem_health_info_count": 1
}
```

`total`, `valid` and `invalid` count documents only, so `summary.total` agrees with
`query --count` on the same path. Schema hygiene is counted on its own axis.

### Upgrading from version 1

| Version 1 | Version 2 |
|-----------|-----------|
| `validate <file>` emitted a bare `rootline/validate` object | Read `.results[0]`; `--field valid` becomes `--field "results[].valid"` |
| `.stem` findings appeared in `results[]` with `source: "stem-health"` | Read `.stem_health[]`, keyed by `check` |
| Directory structural verdicts appeared in `results[]` with a trailing-slash path | Read `.structural[]` |
| `summary.total` counted records plus health findings | `summary.total` counts records |
| `validate --staged` wrote nothing on an empty index | Emits the envelope with `summary.total: 0` |
| A missing or unparseable `.stem` wrote a Go error to stderr | Emits the envelope with a `scan_failed` notice, still exit 1 |

## Section Validation

When a `.stem` defines `type: section` fields, Rootline validates their presence alongside frontmatter fields during Document Validation (phase 2).

Required sections are checked like required frontmatter fields. A missing section emits an error with `rule: required` and the heading as the `field` value:

```json
{
  "rule": "required",
  "field": "## Summary",
  "message": "required section \"## Summary\" is missing",
  "source": "docs/.stem",
  "severity": "error"
}
```

Section validation works in both single-file (`validate <file>`) and batch (`validate --all`) modes. Sections are matched by their heading text, normalized to trim leading/trailing whitespace.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | No errors on any axis |
| `1` | An invalid record, a structural **error**, a `.stem` health **error**, or an **error** notice — plus, under `--strict`, any warning on any of those axes |

`info`-level stem health never affects the exit code.
