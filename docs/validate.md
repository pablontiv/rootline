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

1. **Stem Health** — 12 diagnostics on `.stem` files themselves:
   - `stem-files-exist` — the scanned tree contains at least one `.stem` file
   - `yaml-valid` — valid YAML syntax
   - `scope-match` — scope patterns match at least one file
   - `type-consistency` — field types are consistent across hierarchy
   - `enum-values` — enums have at least 2 values
   - `rule-field-exists` — validation rules reference defined fields
   - `field-override` — child field overrides warn about partial override
   - `aggregated-required` — warns when a field is both `required` and aggregated (`required` is auto-skipped on index files, so the combination rarely does what it looks like)
   - `aggregate-formula-coverage` — an aggregate formula references every enum value of the field it aggregates
   - `monotonic-violations` — child constraints do not widen parent constraints (type, required, enum, severity, structural)
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

## Single File Result

```json
{
  "version": 1,
  "kind": "rootline/validate",
  "path": "docs/query.md",
  "valid": true,
  "errors": [],
  "warnings": []
}
```

When validation fails, errors include the rule, field, message, source `.stem`, severity, and an optional `suggestion` (fuzzy "did you mean?" hint):

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

## Batch Result

```json
{
  "version": 1,
  "kind": "rootline/validate-batch",
  "results": [ ... ],
  "drift_warnings": [
    {
      "field": "estado",
      "parent_value": "Completed",
      "children_value": "Pending",
      "parent_path": "docs/epics/E03/README.md",
      "child_paths": ["docs/epics/E03/F05/README.md"]
    }
  ],
  "summary": {
    "total": 42,
    "valid": 40,
    "invalid": 2,
    "errors_count": 3,
    "warnings_count": 0,
    "drift_warnings_count": 1
  }
}
```

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
| `0` | All documents valid |
| `1` | Errors found (or warnings when `--strict`) |
