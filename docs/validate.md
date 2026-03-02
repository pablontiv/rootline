---
estado: Completed
---
# Validation Engine

`rootline validate` checks documents against the effective schema defined by `.stem` files in the directory tree. It supports single-file, batch, and git-staged modes.

## CLI Usage

```bash
rootline validate docs/api/overview.md           # Single file
rootline validate --all                           # All files in scope
rootline validate --all docs/epics/               # All files under path
rootline validate --staged                        # Git staged files only
rootline validate --all --strict                  # Warnings as errors
rootline validate --all --where 'estado != "Completed"'  # Filtered
```

### Flags

| Flag | Description |
|------|-------------|
| `--all` | Validate all files in scope from current directory |
| `--staged` | Validate only files in git staging area |
| `--strict` | Treat warnings as errors (exit code 1) |
| `--where "expr"` | Filter records in `--all` mode (expr-lang syntax, repeatable) |

## Validation Phases (--all mode)

Batch validation runs four phases in order:

1. **Stem Health** — 8 diagnostics on `.stem` files themselves: yaml-valid, scope-match, type-consistency, enum-values, rule-field-exists, field-override, aggregated-required, version-deprecated.
2. **Document Validation** — Checks each record against its effective schema: required fields, enum values, non_empty, exists, requires rules.
3. **Structural Validation** — Directory-level rules: `require_index` (must have README.md), `min_children`/`max_children` constraints.
4. **Drift Detection** — Warns when an index file's field value contradicts its children (e.g., parent says "Completed" but children are "Pending").

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

When validation fails, errors include the rule, field, message, source `.stem`, and severity:

```json
{
  "version": 1,
  "kind": "rootline/validate",
  "path": "docs/draft.md",
  "valid": false,
  "errors": [
    {
      "rule": "required",
      "field": "estado",
      "message": "required field \"estado\" is missing",
      "source": "docs/.stem",
      "severity": "error"
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

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All documents valid |
| `1` | Errors found (or warnings when `--strict`) |
