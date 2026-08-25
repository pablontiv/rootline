# Validation and Repair Reference

## validate

Use `validate` to check Markdown records against the effective `.stem` schema.

### Target Rules

| User target | Command |
|---|---|
| file(s) | `rootline validate <file...> -o json` |
| directory | `rootline validate --all <dir> -o json` |
| repository scope | `rootline validate --all -o json` |
| staged files | `rootline validate --staged -o json` |

`validate` exits non-zero when validation errors exist. If JSON was requested, parse stdout
anyway — every invocation emits the envelope, including the failure paths.

### Frontmatter Scope

Only the **leading** `---`-delimited block is frontmatter. Everything after its closing
delimiter is Markdown body:

- `---`, `***`, and `___` thematic breaks in the body are ordinary content. They are never
  read as YAML document separators, no matter how many appear.
- Content inside fenced code blocks (` ``` ` and `~~~`) is never parsed as frontmatter or as
  a delimiter.
- A file whose first line is `---` opens a frontmatter block (Jekyll/Hugo convention), even
  when the author meant a thematic break. Put a heading or a blank line first to avoid this.

Two structural rules can fire on the frontmatter block itself:

| Rule | Fires when | Fix |
|---|---|---|
| `malformed_yaml` | the leading block is unterminated, or its YAML does not parse | close the block with `---`; quote values containing `:` |
| `multiple_yaml_documents` | the frontmatter region holds more than one YAML document (a column-0 `...` or `---` marker followed by further content) | keep a single document in the block |

Neither rule inspects the body. Do not advise rewriting body `---` separators as `***` — that
workaround is obsolete.

### Flags

| Flag | Use |
|---|---|
| `--all` | scan all records under the target directory |
| `--staged` | validate git-staged Markdown files |
| `--strict` | treat warnings as errors |
| `--where "expr"` | filter records in `--all` mode |

### JSON Shape

One envelope for every invocation — single file, multiple files, `--all`, `--staged` with an
empty index, and the scan-failure path. Never branch on the flags; the keys are always present.

```json
{
  "version": 2,
  "kind": "rootline/validate-batch",
  "results": [
    { "version": 1, "kind": "rootline/validate", "path": "file.md",
      "valid": false, "errors": [], "warnings": [] }
  ],
  "structural": [
    { "version": 1, "kind": "rootline/validate", "path": "sub/",
      "valid": true, "errors": [], "warnings": [] }
  ],
  "stem_health": [
    { "path": "sub/.stem", "check": "scope-match",
      "severity": "warn", "message": "scope.match \"*.txt\" matches no files in directory" }
  ],
  "drift_warnings": [],
  "notices": [ { "severity": "warn", "code": "no_records", "message": "no records found in scope" } ],
  "summary": {
    "total": 10, "valid": 8, "invalid": 2,
    "errors_count": 3, "warnings_count": 1, "drift_warnings_count": 0,
    "structural_errors_count": 0, "structural_warnings_count": 0,
    "stem_health_errors_count": 0, "stem_health_warnings_count": 1, "stem_health_info_count": 0
  }
}
```

Reading it:

- A single-file verdict is `.results[0]`, not the top level. `--field valid` is now
  `--field "results[].valid"`.
- `results` holds documents only. `summary.total` is a record count and agrees with
  `query --count` on the same path. Directory verdicts from `structural:` rules are in
  `structural[]` (trailing-slash paths; the scan root is `"/"`).
- `.stem` diagnostics are in `stem_health`, keyed by `check`, with `severity` `error`,
  `warn` or `info`. `info` (e.g. `nested-root-marker`) never fails `--strict`.
- `notices` carries run-level diagnostics by stable `code`: `scan_failed`,
  `schema_resolution_failed`, `stem_health_unavailable`, `no_records`.
- A tree with no `.stem`, or one that does not parse, still emits this envelope —
  `stem-files-exist` / `yaml-valid` in `stem_health`, `scan_failed` in `notices`, exit 1.
- A single file target whose `.stem` is missing or unparseable, and a chain where no
  `.stem` declares the boundary, also emit the envelope: `schema_resolution_failed` in
  `notices`, `stem_health` empty, exit 1. Parse stdout on these paths too.
- A file target and `--all` reach the same verdict on the same tree. Naming a file never
  relaxes governance, so validating changed files in a hook agrees with validating the
  tree in CI.
- Declining the interactive root-marker prompt refuses the command rather than running it,
  so a terminal and CI reach the same verdict on the same tree.
- `validate --staged` on an empty index emits the envelope with `summary.total: 0`, exit 0.

Each issue in `results[]` includes `rule`, `field`, `message`, `source`, `severity`, and optional `suggestion`.

Link-check rules (emitted when the effective `.stem` sets `links.checks`): `link_resolve` (target missing, case-sensitive; carries fuzzy `suggestion`; wikilinks infer `.md` so `[[b]]` matches `b.md` and `[[sub/README]]` matches `sub/README.md`, while markdown targets resolve literally; root-anchored `/x.md` resolves against the scan root, or the governance boundary for single-file `validate`; resolution is clamped to that root, so a target escaping it via `..` or via a symlink pointing outside the tree never resolves, while a symlink staying inside still does), `link_anchor` (`#anchor` matches no heading slug in the target), `link_encoding` (raw space in target; use `%20`). These are not auto-fixable by `fix` — repair the link or the target file manually. `checks.cycles: true` additionally makes `graph --check` fail on link cycles; without it cycles are printed as informational and only broken links set the exit code (override per-run with `--fail-cycles`).

### Body-Sourced Field Validation

Schema fields with `source:` directives (body-extracted fields) now participate in validation. The `required` and `enum` constraints apply to values extracted from the document body:

**Extraction directives**:
- `source: body.h1` — extracts the text of the first H1 heading (e.g., `# My Document` → `"My Document"`)
- `source: body.section["## Heading"]` — extracts content under the named section (e.g., `## Notes` with content below it)

**Precedence**: Frontmatter takes absolute precedence. If a field key exists in the record's YAML frontmatter, that value is used and body extraction is skipped.

**Constraint application**:
- `required means presence`: a missing section fails, while an empty section is present with value `""`
- `values: [values...]` with `source:` validates the extracted value against the allowed list
- `non_empty` is a separate content constraint; both checks use the same Phase 1 resolver

**This can turn a passing document into a failing one.** Before, a body-sourced field
resolved to nothing, so its `enum` never ran. Now the extracted text is checked. A `.stem`
pairing `values:` with `source: body.section[...]` over free-form prose will start reporting
`enum` errors on documents that validated cleanly before. Either drop `values:` from the
field, widen the list, or set `severity: off` on it.

**Example**:

```yaml
# .stem
schema:
  notes:
    type: string
    required: true
    source: body.section["## Notes"]
  status:
    type: enum
    values: [approved, pending, rejected]
    source: body.section["## Status"]
```

In validation:
- Document with `## Notes` section + content → passes `required`
- Document without section → fails `required`
- Document with `## Status` containing "approved" → passes `enum`
- Document with "invalid" in that section → fails `enum`
- Document with frontmatter `notes: "value"` and no body section → passes (frontmatter wins)

### Source and provenance

`body.section[...]` matches an exact heading level and text. Duplicate matching headings fail rather than choosing an occurrence; frontmatter remains an override. Path-like validation error sources are governance-root-relative, while symbolic sources stay symbolic.

### Reporting Format

Use this exact shape in responses:

```text
<path>
  ERROR <field>: <message> (rule: <rule>, source: <source>)
  WARN  <field>: <message> (rule: <rule>, source: <source>)

Summary: <valid>/<total> valid | <errors> errors | <warnings> warnings
```

Read file frontmatter only when an issue needs local context that the JSON does not provide.

## fix

Use `fix` to apply Rootline proposals. Always preview first.

### Target Rules

| User target | Preview | Apply |
|---|---|---|
| file(s) | `rootline fix <file...> --dry-run` | `rootline fix <file...>` |
| directory or repository scope | `rootline fix --all <dir> --dry-run -o json` | `rootline fix --all <dir>` |

`fix --all --dry-run -o json` returns proposal JSON. Single-file `fix` prints human text.

### Flags

| Flag | Use |
|---|---|
| `--all` | fix all records under target directory |
| `--dry-run` | preview without writing |
| `--no-propagate` | skip aggregate propagation proposals |

### Batch Preview JSON

```json
{
  "version": 1,
  "kind": "rootline/proposals",
  "path": "docs/",
  "root": "/abs/path/to/docs",
  "proposals": null,
  "schema_suggestions": [
    { "type": "extend_enum", "field": "estado", "paths": ["a.md"], "value": "Blocked" }
  ],
  "summary": {
    "total": 1,
    "extend_enum": 1,
    "migrate_value": 0,
    "correct_value": 0,
    "extract_body": 0,
    "infer_from_children": 0,
    "add_field": 0,
    "infer_from_siblings": 0,
    "correct_outlier": 0,
    "correct_link": 0,
    "add_aggregate": 0,
    "remove_stem_field": 0,
    "propagate_aggregate": 0,
    "schema_evolution": 0,
    "remove_field": 0,
    "loosen_required": 0,
    "change_type": 0,
    "replace_enum_values": 0,
    "loosen_severity": 0
  }
}
```

`proposals` is `null` when no data repairs are available. `schema_suggestions` is an array in dry-run JSON when schema work is withheld; apply JSON reports it as an integer count. `type_findings` is omitted when empty; when present, treat it as unresolved validation work.

### Batch Apply JSON

```json
{
  "version": 1,
  "kind": "rootline/fix-batch",
  "results": [],
  "summary": { "total": 10, "fixed": 3, "skipped": 7 }
}
```

A `correct_value` proposal with `from_representation` is a representation-only repair.
Its `from` and `to` preserve the same exact scalar text; applying it quotes the YAML value.
Only timestamp, boolean and integer to string are automatic. Treat `type_findings` as
unresolved validation defects and run `validate` after repair even though findings do not
change the `fix` exit code.

### Required Repair Loop

```bash
rootline validate --all <dir> -o json
rootline fix --all <dir> --dry-run -o json
rootline fix --all <dir>
rootline validate --all <dir> -o json
git diff -- <dir>
```

For a file target, replace each `--all <dir>` command with the file form shown above.

### Mutation Scope

`fix --all` writes document frontmatter only through data-repair proposals such as `add_field`, `correct_value`, `migrate_value`, `extract_body`, `set_field`, `set_section`, and `propagate_aggregate`. It never writes `.stem` files; schema-surface work (`extend_enum`, `remove_stem_field`, `add_aggregate`) is withheld in `schema_suggestions` for `schema apply` or manual review. Report document-mutation scope before applying unless the user already authorized repairs.

### Missing required fields are reported, not invented

An `add_field` proposal carries `value_source`:

| `value_source` | Meaning | Applied by default |
|---|---|---|
| `schema_default` | the field's declared `default:` | yes |
| `enum_first` | first member of `values`, no `default:` declared | no |
| `empty` | empty string, nothing declared at all | no |
| absent | report predates provenance | yes (treated as `schema_default`) |

A required field whose schema declares no `default:` is left alone and the skip is
reported. Do NOT read that as a failure: the validation error survives on purpose,
because writing a guessed value would erase the missing-data signal the schema author
asked for. Pass `--fill-missing` only when the user explicitly wants engine-chosen
values written.

Never "fix" a lingering `required field ... is missing` error by editing the document
yourself to make validation pass. Surface it and ask what the value should be.
