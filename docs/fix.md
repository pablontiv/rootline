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
| **migrate_value** | Extracts structured data from free-text invalid values | "Pending (blocked by T001)" → `[[blocks:T001]]` |
| **extract_body** | Finds `**Key**: Value` patterns in the document body | Extracting status from a legacy README |
| **infer_from_siblings** | Uses statistical majority of a directory to fill missing values | Setting `tipo: software` because 90% of files are software |
| **correct_outlier** | Identifies values that differ from a strong consensus in a folder | Flagging a task as "manual" in a "deploy" folder |
| **infer_from_children** | Rolls up status from child records to an index file | README becomes "Completed" if all tasks are done |

## CLI Usage

### Dry Run (Recommended)

Always preview changes before applying them.

```bash
rootline fix --all --dry-run
```

The output will group proposals by type, showing how many files are affected and the rationale behind each suggestion.

### Applying Fixes

```bash
rootline fix --all
```

Rootline applies proposals in order of priority (e.g., updating the `.stem` first, then fixing the data).

## YAML AST Preservation

When Rootline modifies a `.stem` or a frontmatter block, it uses a YAML AST (Abstract Syntax Tree) parser. This **preserves your comments and formatting** while updating the data.

## Sibling Inference Logic

To prevent noisy guesses, sibling inference requires:
- A minimum number of siblings (default: 2 for missing, 3 for outliers).
- A strong consensus threshold (default: 60% for missing, 75% for outliers).
