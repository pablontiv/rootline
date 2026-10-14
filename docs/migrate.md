---
estado: Completed
---
# Schema Migration

When `.stem` files change, existing documents may become invalid. `rootline migrate` detects breaking changes and performs bulk migration operations.

Complementary to `rootline fix` — fix corrects data errors, migrate handles schema evolution.

## CLI Usage

```bash
rootline migrate                          # Diff current .stem vs git HEAD
rootline migrate --dry-run                # Report changes without modifying files
rootline migrate --from old.stem          # Compare against specific .stem file
rootline migrate --rename old_field=new   # Rename field across all documents + .stem files
rootline migrate --split                  # Split flat .stem into hierarchical per-level files
rootline migrate --split --dry-run        # Preview split without writing files
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Report changes without modifying files |
| `--from` | Compare against specified `.stem` file instead of git HEAD |
| `--rename old=new` | Rename a field across all documents and `.stem` files |
| `--split` | Split a flat `.stem` into hierarchical `.stem` files per level |
| `--scaffold` | Scaffold missing required source-backed sections into documents using schema defaults |

## Change Detection

Diff mode compares `.stem` files and classifies each change:

| Change Kind | Breaking? | Example |
|-------------|-----------|---------|
| `field_added` | No | New optional field in schema |
| `field_removed` | Yes | Field deleted from schema |
| `type_changed` | Yes | Field type changed (for example string → enum) |
| `source_changed` | Yes | A body-source binding changed or was removed |
| `prefix_changed` / `digits_changed` | Yes | Sequence format changed |
| `match_changed` / `required_match_changed` / `excludes_changed` | Yes | Field applicability changed |
| `enum_value_added` | No | New allowed value |
| `enum_value_removed` | Yes | Value no longer valid |
| `required_added` | Yes | Field now required |
| `required_removed` | No | Field no longer required |
| `default_changed` | No | Default value changed |
| `severity_changed` | Depends | Severity tightened or loosened |
| `rule_added` / `rule_removed` | Depends | Validation rule added or removed |

Sequence and applicability edits stay in the existing `changes[]` array. Their
`before` and `after` strings contain compact, named JSON snapshots, for example
`{"prefix":"T"}` → `{"prefix":"F"}` or `{"match":null}` →
`{"match":{"patterns":["T*"]}}`. Missing values are explicit (`""`, `0`, or
`null`), and map keys are ordered deterministically. These changes are
conservatively breaking because existing records may be governed or interpreted
differently. Table output includes the same before/after values in its messages.
Values YAML can represent but JSON cannot (such as maps with non-string keys)
use deterministic named YAML snapshots instead, without discarding their data.

Conditional validation rules compare `if` and `then` as well as `rule`, `field`,
and `severity`; provenance does not affect identity. Changing only `if` or only
`then` produces the existing `rule_removed` / `rule_added` pair, not a zero diff.
The removed rule's `before` and added rule's `after` contain complete named
snapshots including both `if` and `then`, so the pair exposes the exact edit.
Rule snapshots include explicit `null` values for absent `if` and `then`, including
unconditional rules, so adding or removing the last condition remains observable.
Unchanged rules and attributes produce no changes. The version-1 envelopes and
their existing top-level keys are unchanged.

## Result Shape

`migrate` emits three `kind` values, one per mode:

| `kind` | Produced by | Payload |
|--------|-------------|---------|
| `rootline/migrate-diff` | A single `.stem` comparison | `stem_path`, `changes`, `breaking_count`, `total_count` |
| `rootline/migrate-batch` | Diffing a directory (every `.stem` under the path) | `results` (one `migrate-diff` each) plus a `summary` |
| `rootline/migrate-rename` | `--rename old=new` | `old_field`, `new_field`, `files_updated`, `stems_updated`, `summary` |

A single-stem diff carries `stem_path`, the absolute path of the `.stem` it describes:

```json
{
  "version": 1,
  "kind": "rootline/migrate-diff",
  "stem_path": "/repo/docs/.stem",
  "changes": [
    {
      "kind": "field_removed",
      "field": "prioridad",
      "breaking": true,
      "before": "enum",
      "after": null,
      "message": "field 'prioridad' removed",
      "affected_files": 12
    }
  ],
  "breaking_count": 1,
  "total_count": 1
}
```

Pointing `migrate` at a directory wraps those diffs in a batch envelope — this is
the shape you get from `rootline migrate docs --dry-run -o json`:

```json
{
  "version": 1,
  "kind": "rootline/migrate-batch",
  "results": [
    {
      "version": 1,
      "kind": "rootline/migrate-diff",
      "stem_path": "/repo/docs/.stem",
      "changes": null,
      "breaking_count": 0,
      "total_count": 0
    }
  ],
  "summary": {
    "stems_checked": 1,
    "total_changes": 0,
    "breaking_count": 0
  }
}
```

`changes` is `null`, not `[]`, when a `.stem` has no changes to report.

## Rename Mode

`--rename` updates a field name across all documents and `.stem` files:

```bash
rootline migrate --rename status=estado
```

Updates frontmatter in all affected markdown files and schema definitions in `.stem` files. Operations are logged to `.rootline-migrations` (JSON Lines, append-only).

Document renames match only a plain, column-0 key in the leading YAML frontmatter
block. Nested keys with the same name are left untouched, even when they appear
before the top-level key. Only the matched key's bytes are replaced: values,
comments, indentation, field order, delimiters, the Markdown body, LF/CRLF line
endings, and any leading UTF-8 BOM remain byte-identical. A nested-only occurrence
or a key that does not match this form is not reported in `files_updated`.
`--dry-run` reports the same document matches without writing them. Schema
definitions in `.stem` files continue to use the schema-aware rename path.

Before writing anything, rename checks every document containing the old field
and every `.stem` declaring it. If any of those already contains the destination
key (in frontmatter or `schema`, respectively), the command refuses the entire
rename with an error naming the colliding path and destination key. All documents
and `.stem` files remain byte-identical, and no migration log entry is appended.
`--dry-run` performs the same collision checks and returns the same refusal.
A destination key in a document or `.stem` that does not contain the old field
does not block the rename.

Each generated document or `.stem` is replaced atomically from a sibling staging
file, so a failed write cannot leave that destination truncated. After collision
preflight succeeds, migration runs are still best-effort rather than transactional:
files completed before a later write error are not rolled back.

## Split Mode

`--split` converts a flat `.stem` into hierarchical per-level `.stem` files. It detects directory naming patterns (e.g., `E##-*`, `F##-*`, `S###-*`, `T###-*`) and distributes schema fields by real presence at each level.

```bash
rootline migrate --split docs/epics/       # Split into per-level .stem files
rootline migrate --split --dry-run docs/   # Preview without writing
```

### How it works

1. Scans records and detects hierarchy levels from directory naming patterns
2. Analyzes which fields have values at which levels
3. Fields present at **all levels** stay in the root `.stem`
4. Fields present at **some levels** go to per-level `.stem` files
5. Each level gets a `sequence` id field matching its prefix/digits
6. `derive`, `aggregate`, `links`, `structural`, and `validate` rules are preserved at root

### Example

Given a flat `.stem` with fields used across `E##/F##/S###/T###` directories:

```
docs/epics/.stem              → root fields + E-level sequence + derive/aggregate/links
docs/epics/E03-name/.stem     → F-level sequence + F-only fields
docs/epics/E03-name/F01-x/.stem → S-level sequence + S-only fields
```

Requires at least 2 hierarchy levels to be detected. Use `--dry-run` to preview the split before applying.

## Scaffold Mode

`--scaffold` adds missing required source-backed sections to documents that do not have them. It uses the `default` property of each applicable field in the effective `.stem` to populate inserted content.

```bash
rootline migrate --scaffold docs/epics/          # Add missing required sections
rootline migrate --scaffold --dry-run docs/      # Preview without writing
```

### Content Priority

When inserting a missing section, content is selected in this order:

1. Non-empty `default` value from the source-backed field in the effective `.stem`
2. `"<!-- TODO -->"` fallback when no default is defined

### Example

Given a `.stem` with:

```yaml
schema:
  summary:
    type: string
    source: body.section["## Summary"]
    required: true
    default: "Describe the purpose of this document."
  references:
    type: string
    source: body.section["## References"]
    required: true
```

Running `rootline migrate --scaffold` on a file missing both sections produces:

```
~ docs/epics/E03/README.md
  + ## Summary
    Describe the purpose of this document.
  + ## References
    <!-- TODO -->
```

Sections are inserted at the end of the document body only when they use simple selectors and need materialization. Multiple simple additions are appended in lexical heading order. A frontmatter override or an empty-present section needs no materialization. Ambiguous source resolution and invalid declarations fail rather than becoming a successful no-op.

Each selector component contains one to six `#` characters, one space, and the exact parsed heading text. The `#` characters encode the heading level, not the original Markdown form. Scaffold materializes a simple selector with ATX or Setext syntax that preserves the level and text. Setext syntax is available only for levels 1 and 2. If neither form preserves the selector, scaffold fails before it writes the affected file.

A qualified selector can use `body.section["## Parent"]["### Notes"]`. The components must be contiguous, and the selector matches a contiguous suffix of the heading path. `migrate --scaffold` does not invent ancestor headings. If a required qualified section is absent, scaffold stops before it writes the affected file. Scaffold validates prospective bytes before each affected-file write. It does not guarantee global atomicity across all affected files. Use `--dry-run` to review insertions before applying.

## Schema Evolution

When running under monotonic constraints (O10), some destructive schema changes are rejected as violations:
- Remove a field from schema
- Loosen a required constraint
- Change a field type incompatibly
- Remove enum values
- Reduce severity levels

These changes can still be legitimate during migrations (e.g., field deprecation, schema redesigns). Schema evolution proposals represent these breaking changes as explicit, reviewable operations with migration rationale.

### Evolution Proposal Types

| Type | Surface | Meaning |
|------|---------|---------|
| `remove_field` | migration | Field explicitly removed from schema (may require data repair) |
| `loosen_required` | migration | Required constraint loosened (may accept legacy records) |
| `change_type` | migration | Field type changed incompatibly (may require data migration) |
| `replace_enum_values` | migration | Enum values replaced (affected records need repair or mapping) |
| `loosen_severity` | migration | Validation severity reduced (formerly critical now warning) |
| `schema_evolution` | migration | Generic evolution marker for unlisted breaking changes |

### Using Evolution Proposals

1. **Detect breaking changes**: `rootline migrate [path] --output json` shows breaking changes
2. **Convert to proposals**: Breaking changes can be converted to schema evolution proposals for explicit approval
3. **Apply with care**: Evolution proposals surface as `migration` class, not `schema` class — they require explicit review and approval, not automatic application

Example workflow:
```bash
# Detect breaking changes in .stem (batch envelope: changes live under results[])
rootline migrate docs/roadmap/ --output json | jq '.results[].changes[]? | select(.breaking == true)'

# These breaking changes can be represented as schema_evolution proposals
# and reviewed for explicit approval before application
```

**Key principle**: Never silently apply schema evolution changes — they should be reviewed, tested, and documented with migration notes explaining the rationale.
