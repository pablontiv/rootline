---
name: describe
description: |
  Show the effective .stem schema for a directory using rootline CLI.
  Displays fields, types, required status, enum values, and which .stem file defines each rule.
  This skill should be used when the user says "describe schema", "show schema",
  "que campos necesito", "what fields are required", "what metadata is needed",
  "show me the fields", "stem rules", "describe this directory",
  or wants to understand document structure for a path.
argument-hint: "[directory-or-file]"
allowed-tools: Bash
---

# /describe — Schema Inspector

Show the effective schema for a directory by merging all ancestor `.stem` files.

## Procedure

### 1. Determine target

- If `$ARGUMENTS` is a **directory** → use that directory
- If `$ARGUMENTS` is **empty** → use current working directory
- If `$ARGUMENTS` is a **file** → use its parent directory

### 2. Execute describe

```bash
rootline describe <directory> --output json
```

If `rootline describe` fails (e.g., directory not found, no `.stem` files in hierarchy), report the error and suggest checking that `.stem` files exist in the directory or its ancestors.

### 3. Parse JSON output

The output follows this structure:

```json
{
  "version": 1,
  "kind": "rootline/describe",
  "path": "docs/epics/E01-foo/",
  "applies": [".stem", "../.stem"],
  "schema": {
    "field_name": {
      "type": "string|enum|sequence",
      "required": true,
      "values": ["val1", "val2"],
      "default": "value",
      "severity": "error|warn|off",
      "source": ".stem file path",
      "prefix": "E",
      "digits": 2,
      "next": "E03"
    }
  },
  "validate": [
    { "field": "name", "rule": "required|non_empty|enum|exists|requires", "severity": "error" }
  ],
  "derive": { "computed_field": "expression" },
  "aggregate": { "rollup_field": "expression" },
  "links": { "allowed": ["blocks"], "rules": {} },
  "hints": ["optional usage hints"]
}
```

### 4. Render as markdown table

Present the schema as a table:

```markdown
| Field | Type | Required | Values | Source |
|-------|------|----------|--------|--------|
| estado | enum | yes | Pending, Completed | .stem |
| tipo | enum | yes | feature, historia | ../.stem |
| id | sequence | yes | (next: T005) | .stem |
```

For sequence fields, show the next auto-generated value in the Values column as `(next: VALUE)`.

### 5. Additional sections

If **validation rules** exist (`validate` array is non-empty), add:

```
### Validation Rules
- FIELD: RULE (severity: SEVERITY)
```

If **derived fields** exist (`derive` is non-empty), add:

```
### Derived Fields
- FIELD: `expression`
```

If **aggregations** exist (`aggregate` is non-empty), add:

```
### Aggregations
- FIELD: `expression`
```

### 6. Show .stem inheritance chain

List which `.stem` files apply (from `applies` array):

```
Schema inherited from:
1. path/to/.stem (local)
2. parent/.stem (inherited)
```

If `hints` are present, display them at the end.
