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

Link-check rules (emitted when the effective `.stem` sets `links.checks`): `link_resolve` (target missing, case-sensitive; carries fuzzy `suggestion`), `link_anchor` (`#anchor` matches no heading slug in the target), `link_encoding` (raw space in target; use `%20`). These are not auto-fixable by `fix` — repair the link or the target file manually. `checks.cycles: true` additionally makes `graph --check` fail on link cycles; without it cycles are printed as informational and only broken links set the exit code (override per-run with `--fail-cycles`).

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
