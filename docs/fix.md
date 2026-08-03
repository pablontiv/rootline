---
estado: Completed
---
# Fix & Proposal Engine

The `rootline fix` command goes beyond simple mechanical repairs. It analyzes validation errors and proposes intelligent solutions based on data patterns and heuristics.

## Proposal Types

Rootline categorizes issues into specific proposal types to preserve semantic meaning.

| Type | Logic | Example |
|------|-------|---------|
| **extend_enum** | If N files share an invalid enum value, propose adding it to the `.stem` | Adding "Obsolete" to status |
| **correct_value** | Suggests the closest valid enum value for a typo | "Completd" → "Completed" |
| **migrate_value** | Extracts structured data from free-text invalid values | "Pending (blocked by T001)" → `[[blocks:T001]]` |
| **extract_body** | Finds `**Key**: Value` patterns in the document body | Extracting status from a legacy README |
| **add_field** | Adds a missing required field with a default or inferred value | Adding `estado: Pending` to a file without it |
| **infer_from_siblings** | Uses statistical majority of a directory to fill missing values | Setting `tipo: software` because 90% of files are software |
| **correct_outlier** | Identifies values that differ from a strong consensus in a folder | Flagging a task as "manual" in a "deploy" folder |
| **infer_from_children** | Rolls up status from child records to an index file | README becomes "Completed" if all tasks are done |
| **correct_link** | Fixes broken wiki-links by resolving to the closest valid target | `[[T099]]` → `[[T001]]` |
| **add_aggregate** | Generates aggregate expressions for index files missing them | Adding `estado: len(filter(...))` to README |
| **remove_stem_field** | Removes invalid fields from `.stem` detected by stem health checks | Removing a field that references a non-existent type |
| **propagate_aggregate** | Detects stale aggregate values in index files and corrects them | Updating `completed` count after child status changes |
| **set_field** | Sets a frontmatter field to a specific value via the `rootline set` pipeline | Setting `estado: Completed` after all children are done |
| **set_section** | Sets or replaces a section body via the `rootline set` pipeline | Populating an empty `## Summary` section with inferred content |

## Proposal Struct

Each proposal in the output has the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Proposal type (see table above) |
| `field` | string | Field name the proposal targets |
| `description` | string | Human-readable explanation |
| `paths` | string[] | Affected file paths |
| `heading` | string | For `set_section` proposals: the Markdown heading to mutate |
| `mode` | string | For `set_section` proposals: `set` (replace) or `append` |

`heading` and `mode` are only populated on `set_field` and `set_section` proposals. `set_field` reuses the existing `applySetField` pipeline from `rootline set`.

## Three Repair Paths

Rootline provides **three separate commands** for different repair scenarios. Choose the right one for your workflow:

### 1. `rootline fix` — Built-in Repair (Legacy)

**Deprecated.** Use `rootline repair apply` instead (see below).

```bash
rootline fix --all --dry-run   # Preview proposals (data-only)
rootline fix --all             # Apply data-only repairs
```

`fix` combines proposal generation and application in one step. It applies only **data-only repairs** to frontmatter and skips schema-surface proposals, which appear in the `schema_suggestions` field of the output.

### 2. `rootline repair apply` — Data-Only Bulk Repair (Current)

Use this for fixing frontmatter data issues found by `rootline validate` or `rootline analyze`.

**Workflow:**

```bash
cd docs/

# 1. Generate a proposals report
rootline fix --all . --dry-run -o json > fix-proposals.json

# 2. Review the proposals (schema_suggestions are separated)
# 3. Apply repair proposals (frontmatter only, never .stem)
rootline repair apply --report fix-proposals.json --dry-run
rootline repair apply --report fix-proposals.json
```

Keep the report file in the directory you scanned. `repair apply` resolves the paths in the
report against the **report file's own directory**, so a report written elsewhere (a `reports/`
or CI artifacts directory) resolves to paths that do not exist and the run becomes a no-op.

**Report envelope.** `repair apply` validates the report envelope before touching any file
and rejects anything that is not `version: 1` and `kind: rootline/proposals`:

```bash
rootline repair apply --report analyze.json
# Error: wrong report kind: rootline/analyze (expected rootline/proposals)
```

An `analyze` report is therefore not a valid input here — produce the report with
`rootline fix --all <dir> --dry-run -o json`. For the schema-surface half of an analyze
report, use `rootline schema apply`, which does accept `kind: rootline/analyze`.

`repair apply` applies only repair-surface proposals to document frontmatter:
- correct_value
- add_field
- migrate_value
- extract_body
- infer_from_siblings
- correct_outlier
- infer_from_children
- propagate_aggregate
- set_field
- set_section

Schema proposals (extend_enum, add_aggregate, remove_stem_field) are **silently rejected** — use `rootline schema apply` instead.

**Path containment.** A report is untrusted input, so every path it names is checked against
the scan root before any file is opened. A path that escapes the root — `../../../etc/shadow`
or an absolute path — is refused outright and never read, let alone written. Refusals land in
`rejected[]`, not `errors[]`, keeping a policy decision distinguishable from a failed write:

```bash
rootline repair apply --report fix-proposals.json
# rejected: containment violation: path "../outside.md" escapes root (root "/abs/scan")
```

A proposal is applied whole or not at all, so a single escaping path discards the entire
proposal rather than applying it partially.

With `--dry-run`, the output additionally carries `resolved_targets` so you can see where each
write would have landed and why any were refused before committing to the run:

```json
{
  "resolved_targets": {
    "accepted": { "tasks/T001.md": "/abs/scan/tasks/T001.md" },
    "rejected": { "../outside.md": "escapes root" }
  }
}
```

`resolved_targets` is additive and omitted outside dry-run; the report contract stays `version: 1`.

Flags:
- `--dry-run` — Preview changes without modifying files
- `--report <file>` — Path to proposals JSON file (required)

### 3. `rootline schema apply` — Schema Mutation

Use this for applying schema changes (extending enums, creating `.stem` fields, updating `.stem` files).

**Workflow:**

```bash
# 1. Generate schema proposals from analyze report
rootline analyze docs/ --incremental > analyze.json

# 2. Review schema_suggestions in analyze.json
# 3. Apply schema proposals to .stem files
rootline schema apply --report analyze.json --dry-run
rootline schema apply --report analyze.json

# Or generate proposals without analysis:
rootline schema propose docs/ > proposals.json
rootline schema apply --report proposals.json --dry-run
rootline schema apply --report proposals.json
```

`schema apply` accepts a report from `rootline analyze` or `rootline schema propose` and applies only schema-surface proposals to `.stem` files:
- create_stem
- update_stem
- extend_enum

Data-only repairs (correct_value, add_field, etc.) are **ignored** — use `rootline repair apply` instead.

**Path containment.** As with `repair apply`, a report is untrusted input: every `create_stem`
target is checked against the scan root before a `.stem` is written, so a report naming
`<root>/../../outside/.stem` no longer scaffolds outside the tree the command was pointed at.

`schema propose` emits **absolute** targets, so the propose→apply loop depends on absolute paths
staying valid — they are accepted and then confined, not refused outright. That is the one
difference from `repair apply`, which rejects absolute paths as malformed input.

The second difference is where refusals land. `schema apply` has no `rejected[]`, so a target
outside the root is a validation failure and appears in `errors[]`:

```bash
rootline schema apply --report proposals.json
# errors: containment violation: path "/tmp/outside/.stem" escapes root (root "/abs/scan")
```

With `--dry-run`, the output carries the same additive `resolved_targets` envelope that
`repair apply` emits, so you can see where each `.stem` would land before writing:

```json
{
  "resolved_targets": {
    "accepted": { "/abs/scan/tasks/.stem": "/abs/scan/tasks/.stem" },
    "rejected": { "/tmp/outside/.stem": "escapes root" }
  }
}
```

`resolved_targets` is omitted outside dry-run; the contract stays `version: 1`.

Flags:
- `--dry-run` — Preview changes without modifying files
- `--report <file>` — Path to proposals JSON file (required)

## When to Use Each

| Scenario | Command | Example |
|----------|---------|---------|
| Add missing required fields to documents | `repair apply` | Missing `estado` field in all tasks |
| Correct misspelled enum values | `repair apply` | `"Complted"` → `"Completed"` |
| Extend `.stem` enum to accept new valid values | `schema apply` | Adding `"Archived"` to allowed statuses |
| Create `.stem` for a new directory | `schema apply` | Initialize schema after inferring patterns |
| Update `.stem` after schema changes | `schema apply` | Add new field definitions after inference |

## YAML AST Preservation

When Rootline modifies a `.stem` or a frontmatter block, it uses a YAML AST (Abstract Syntax Tree) parser. This **preserves your comments and formatting** while updating the data.

## Sibling Inference Logic

To prevent noisy guesses, sibling inference requires:
- A minimum number of siblings (default: 2 for missing, 3 for outliers).
- A strong consensus threshold (default: 60% for missing, 75% for outliers).
