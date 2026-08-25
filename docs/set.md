---
estado: Completed
---
# Set Command

`rootline set` mutates frontmatter fields with schema validation. It is the primary mutation command — changes are validated against the effective `.stem` schema before and after application, with automatic rollback on failure.

## Synopsis

```bash
rootline set <file> <field=value> [<field=value>...] [flags]
```

## Arguments

| Argument | Description |
|----------|-------------|
| `file` | Path to the markdown file to mutate |
| `field=value` | One or more assignment expressions (see Operators) |

## Operators

| Operator | Syntax | Description |
|----------|--------|-------------|
| Set | `field=value` | Set a frontmatter field to the given value |
| From file | `field=@path` | Read value from the file at `path` and set it |

For frontmatter fields, `field` is the YAML key name.

## Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Show proposed changes without writing to disk |
| `--no-validate` | Skip post-mutation validation (pre-validation of enum constraints always runs) |

## Pipeline

`rootline set` runs the following steps in order:

1. **Parse** — Load the file and parse frontmatter + body
2. **Extract** — Run the full extraction pipeline (AST, sections map)
3. **Schema** — Resolve the effective `.stem` for the file's directory
4. **Pre-validate** — Check proposed values against schema constraints (enum membership, type)
5. **Apply** — Write mutations to the in-memory document
6. **Post-validate** — Run full validation on the mutated document
7. **Rollback** — If post-validation fails and `--no-validate` is not set, discard changes and report errors

If `--dry-run` is active, the pipeline stops before step 6 and prints the proposed diff.

## Examples

### Set a frontmatter field

```bash
rootline set docs/api/overview.md estado=Completed
```

### Set multiple fields at once

```bash
rootline set docs/api/overview.md estado=Completed tipo=documentation
```

### Set value from a file

```bash
rootline set docs/api/overview.md descripcion=@notes.txt
```

### Preview changes without writing

```bash
rootline set docs/api/overview.md estado=Pending --dry-run
```

### Skip post-validation (advanced)

`--no-validate` skips post-mutation validation only; enum/type pre-validation still runs. Use it only when you deliberately need to write a schema-valid value while leaving some other existing validation error unresolved.

```bash
rootline set docs/api/overview.md estado=review --no-validate
```

## Error Messages

| Situation | Message |
|-----------|---------|
| Invalid enum value | `Error: value "Custom" not in allowed values for estado: [draft, review, published]` |
| Post-validation failure | `Error: post-mutation validation failed (rolled back): tipo: required field "tipo" is missing` |
| File not found | `Error: file not found: /abs/path/docs/api/missing.md (use 'rootline new' to scaffold a new document)` |
| Source-backed field | `set` writes a frontmatter override and does not edit the body section |
| Source file not found (`=@path`) | `Error: resolving value for notes: reading notes.txt: open notes.txt: no such file or directory` |

## Dry Run Output

With `--dry-run`, `rootline set` prints the proposed assignment and exits before writing:

```console
$ rootline set docs/api/overview.md estado=Completed --dry-run
would set estado = "Completed"
```

## Source-Backed Fields

For `source: body.section["## Notes"]`, `set notes=override` writes a frontmatter override. Validation, query, describe, and explain all resolve that override before body extraction; an empty-present section remains present when no override exists.

## Notes

- YAML node rewriting preserves comments and key order where possible, but normalizes inter-token whitespace, inline-comment spacing, and nested indentation.
