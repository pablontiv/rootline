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

## CLI Usage

### Dry Run (Recommended)

Always preview changes before applying them.

```bash
rootline fix --all --dry-run
rootline fix --all --no-propagate   # Skip aggregate propagation proposals
```

The output groups proposals by type, showing affected files and rationale:

```json
{
  "version": 1,
  "kind": "rootline/proposals",
  "proposals": [
    {
      "type": "extract_body",
      "field": "fecha",
      "description": "extract \"2026-02-25\" from body to frontmatter",
      "paths": ["research/kedral/README.md"]
    },
    {
      "type": "add_field",
      "field": "estado",
      "description": "required field \"estado\" is missing",
      "paths": ["research/kedral/design.md", "research/kedral/roadmap.md"]
    }
  ],
  "schema_suggestions": [
    {
      "type": "extend_enum",
      "field": "estado",
      "description": "2 records use \"Archivado\"; propose adding to enum",
      "paths": [".stem"]
    }
  ],
  "summary": {
    "total": 3,
    "extend_enum": 1,
    "migrate_value": 0,
    "correct_value": 0,
    "extract_body": 1,
    "infer_from_children": 0,
    "add_field": 2,
    "infer_from_siblings": 0,
    "correct_outlier": 0,
    "correct_link": 0,
    "add_aggregate": 0,
    "remove_stem_field": 0,
    "propagate_aggregate": 0
  }
}
```

### Applying Fixes

```bash
rootline fix --all
```

`fix --all` applies **data-only repairs** (correct_value, add_field, migrate_value, extract_body, etc.) to document frontmatter. It does NOT apply schema-surface proposals (extend_enum, add_aggregate, remove_stem_field) — these appear as suggestions in the `schema_suggestions` array.

Schema proposals require careful review and are meant to be applied separately:

```bash
# To apply schema changes, manually edit .stem files or use:
rootline describe --by-domain  # Review current schema
# Then manually update .stem based on schema_suggestions
```

## YAML AST Preservation

When Rootline modifies a `.stem` or a frontmatter block, it uses a YAML AST (Abstract Syntax Tree) parser. This **preserves your comments and formatting** while updating the data.

## Sibling Inference Logic

To prevent noisy guesses, sibling inference requires:
- A minimum number of siblings (default: 2 for missing, 3 for outliers).
- A strong consensus threshold (default: 60% for missing, 75% for outliers).
