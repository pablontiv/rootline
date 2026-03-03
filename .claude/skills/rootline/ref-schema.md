# Schema Inspection & Document Scaffolding Reference

## describe — Show effective schema for a directory

### Usage

```bash
rootline describe <path> --output json              # Full schema
rootline describe <path> --output table             # Human-readable table
rootline describe <dir> --field schema.id.next      # Extract next auto-ID
rootline describe <dir> --field schema.estado.values # Extract enum values
```

### Procedure

1. **Determine target**:
   - Directory → use directly
   - File → use its parent directory
   - Empty → use current working directory

2. **Execute**: `rootline describe <directory> --output json`

3. **Parse JSON output** (`kind: "rootline/describe"`):

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

4. **Render as markdown table**:

```markdown
| Field | Type | Required | Values | Source |
|-------|------|----------|--------|--------|
| estado | enum | yes | Pending, Completed | .stem |
| tipo | enum | yes | feature, historia | ../.stem |
| id | sequence | yes | (next: T005) | .stem |
```

For sequence fields, show `(next: VALUE)` in the Values column.

5. **Additional sections** (if present):

- **Validation Rules** (`validate` array): `FIELD: RULE (severity: SEVERITY)`
- **Derived Fields** (`derive` map): `FIELD: expression`
- **Aggregations** (`aggregate` map): `FIELD: expression`

6. **Show .stem inheritance chain** (from `applies` array):

```
Schema inherited from:
1. path/to/.stem (local)
2. parent/.stem (inherited)
```

Display `hints` at the end if present.

---

## new — Scaffold a document with frontmatter

### Usage

```bash
rootline new <filepath>            # Create document
rootline new <filepath> --dry-run  # Preview without writing
rootline new <filepath> --force    # Overwrite existing
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Show generated content without writing file |
| `--force` | Overwrite existing file |

### Procedure

1. **Determine target**:
   - File path (ends in `.md`) → use as target directly
   - Directory → build filename using auto-numbering (step 2)
   - Empty → run `rootline tree . --output table` to show directories, then `AskUserQuestion` to pick one

2. **Auto-numbering** (if target is a directory):

```bash
rootline describe <directory> --field schema.id.next
```

Returns next ID (e.g., `"T005"`, `"E03"`). Build filename as `<ID>-<name>.md`. If no name given, ask the user.

3. **Preview**: `rootline new <filepath> --dry-run`

4. **Create**: `rootline new <filepath>`

The command:
- Scaffolds frontmatter from effective `.stem` schema
- Sets enum fields to first allowed value
- Uses defaults where defined
- Generates a title from the filename

5. **Validate**: `rootline validate <filepath> --output json`

6. **Suggest next steps**: Edit required fields, run `/rootline validate`, use `/rootline describe` for field reference.

### Typical workflow

```bash
rootline describe docs/tasks/ --field schema.id.next   # → "T005"
rootline new docs/tasks/T005-implement-feature.md --dry-run
rootline new docs/tasks/T005-implement-feature.md
rootline validate docs/tasks/T005-implement-feature.md
```
