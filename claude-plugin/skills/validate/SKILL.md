---
name: validate
description: |
  Validate documents against .stem schemas using rootline CLI.
  Checks frontmatter fields for required values, enum constraints, and structural rules.
  Use when the user says "validate", "check schema", "validar", "check this file",
  or wants to verify documents comply with their .stem definitions.
argument-hint: "[file-or-directory]"
allowed-tools: Bash, Read
---

# /validate — Document Validation

Validate documents against their effective `.stem` schema using `rootline validate`.

## Procedure

### 1. Determine target

- If `$ARGUMENTS` is a **file path** → validate that single file
- If `$ARGUMENTS` is a **directory** → validate all files in that directory with `--all`
- If `$ARGUMENTS` is **empty** → validate all files from current directory with `--all`

### 2. Execute validation

Run the appropriate command via Bash:

```bash
# Single file
rootline validate <file> --output json

# Directory or all
cd <directory> && rootline validate --all --output json
```

### 3. Parse and present results

The JSON output has two possible shapes:

**Single file** (`kind: "rootline/validate"`):
```json
{
  "version": 1,
  "kind": "rootline/validate",
  "path": "file.md",
  "valid": true,
  "errors": [],
  "warnings": []
}
```

**Multiple files** (`kind: "rootline/validate-batch"`):
```json
{
  "version": 1,
  "kind": "rootline/validate-batch",
  "results": [ ... ],
  "summary": { "total": N, "valid": N, "invalid": N, "errors_count": N, "warnings_count": N }
}
```

Each error/warning has: `rule`, `field`, `message`, `source` (.stem path), `severity`.

### 4. Format output

**If valid** (no errors):

```
Document is valid. No errors found.
```

For batch with all valid:

```
All N documents are valid. No errors found.
```

**If errors found**, present each error as:

```
PATH
  ERROR  field: message (rule: RULE, source: .stem-path)
  WARN   field: message (rule: RULE, source: .stem-path)
```

End with summary for batch:

```
Summary: N/TOTAL valid | ERRORS errors | WARNINGS warnings
```

### 5. Suggest fixes

If errors are found, suggest running `rootline fix <path>` to auto-correct fixable issues.
