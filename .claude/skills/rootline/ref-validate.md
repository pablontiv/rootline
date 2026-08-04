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

`validate` exits non-zero when validation errors exist. If JSON was requested, parse stdout anyway.

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

### JSON Shapes

Single file:

```json
{
  "version": 1,
  "kind": "rootline/validate",
  "path": "file.md",
  "valid": false,
  "errors": [],
  "warnings": []
}
```

Batch:

```json
{
  "version": 1,
  "kind": "rootline/validate-batch",
  "results": [],
  "summary": { "total": 10, "valid": 8, "invalid": 2, "errors_count": 3, "warnings_count": 1 }
}
```

Each issue includes `rule`, `field`, `message`, `source`, `severity`, and optional `suggestion`.

Link-check rules (emitted when the effective `.stem` sets `links.checks`): `link_resolve` (target missing, case-sensitive; carries fuzzy `suggestion`; wikilinks infer `.md` so `[[b]]` matches `b.md` and `[[sub/README]]` matches `sub/README.md`, while markdown targets resolve literally; root-anchored `/x.md` resolves against the scan root, or the governance boundary for single-file `validate`; resolution is clamped to that root, so a target escaping it via `..` or via a symlink pointing outside the tree never resolves, while a symlink staying inside still does), `link_anchor` (`#anchor` matches no heading slug in the target), `link_encoding` (raw space in target; use `%20`). These are not auto-fixable by `fix` — repair the link or the target file manually. `checks.cycles: true` additionally makes `graph --check` fail on link cycles; without it cycles are printed as informational and only broken links set the exit code (override per-run with `--fail-cycles`).

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
  "proposals": [],
  "summary": {}
}
```

### Batch Apply JSON

```json
{
  "version": 1,
  "kind": "rootline/fix-batch",
  "results": [],
  "summary": { "total": 10, "fixed": 3, "skipped": 7 }
}
```

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

`fix --all` may update documents and `.stem` files through proposals such as `add_field`, `correct_value`, `migrate_value`, `extend_enum`, `remove_stem_field`, `add_aggregate`, and `propagate_aggregate`. Report this scope before applying unless the user already authorized repairs.

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
