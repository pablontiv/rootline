# Validation & Fix Reference

## validate — Check documents against .stem schemas

### Usage

```bash
rootline validate [file...]                              # Specific files
rootline validate --all                                  # All files in scope
rootline validate --all --where "estado == 'Pending'"    # Filtered
rootline validate --staged                               # Git staging area only
rootline validate --strict                               # Warnings → errors
```

### Flags

| Flag | Description |
|------|-------------|
| `--all` | Validate all files in scope from current directory |
| `--staged` | Validate only files in git staging area |
| `--strict` | Treat warnings as errors (exit code 1) |
| `--where "expr"` | Filter expression for `--all` mode (repeatable) |

### Procedure

1. **Determine target**:
   - File path → validate that single file
   - Directory → use `--all` with directory as positional arg
   - Empty → use `--all` from current directory

2. **Execute**: `rootline validate <target> --output json`

3. **Parse JSON output**:

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
  "summary": { "total": 10, "valid": 8, "invalid": 2, "errors_count": 3, "warnings_count": 1 }
}
```

Each error/warning has: `rule`, `field`, `message`, `source` (.stem path), `severity`.

4. **Present results**:

If valid:
```
All N documents are valid. No errors found.
```

If errors:
```
PATH
  ERROR  field: message (rule: RULE, source: .stem-path)
  WARN   field: message (rule: RULE, source: .stem-path)

Summary: N/TOTAL valid | ERRORS errors | WARNINGS warnings
```

5. **Show context**: Use `Read` to show frontmatter alongside errors so user sees current values.

6. **Suggest fixes**: If errors are fixable, suggest `rootline fix <path>`.

---

## fix — Auto-repair validation errors

### Usage

```bash
rootline fix [file...]          # Fix specific files
rootline fix --all              # Fix all files in scope
rootline fix --dry-run          # Preview changes without writing
rootline fix --all --dry-run    # Preview all fixes
```

### Flags

| Flag | Description |
|------|-------------|
| `--all` | Fix all files in scope from current directory |
| `--dry-run` | Show proposed changes without modifying files |

### Procedure

1. Run `rootline fix <target> --dry-run --output json` first to preview
2. Show the user what will change
3. If user approves, run without `--dry-run`

### Output format

**Single file**: Console messages like `"file: added X"`, `"file: corrected Y"`

**Batch** (`kind: "rootline/fix-batch"`):
```json
{
  "version": 1,
  "kind": "rootline/fix-batch",
  "results": [...],
  "summary": { "total": 10, "fixed": 3, "skipped": 7 }
}
```

### Proposal types

The fix engine generates typed proposals:
- `add_field` — add missing required field
- `extend_enum` — add value to .stem enum
- `correct_value` — fix invalid enum value
- `migrate_value` — update deprecated value
- `remove_stem_field` — clean up unused .stem field
- `add_aggregate` — add missing aggregation

### Typical workflow

```bash
rootline validate --all --output json    # Find errors
rootline fix --all --dry-run             # Preview fixes
rootline fix --all                       # Apply fixes
rootline validate --all                  # Verify
```
