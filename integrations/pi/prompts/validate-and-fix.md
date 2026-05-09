# Validate and Fix Records

**Description**: Validate all records in a directory and apply automatic fixes to resolve validation errors.

**Arguments**:
- `directory`: Path to the directory containing Markdown records (e.g., `docs/epics/`, `.`)
- `dry-run` (optional): Preview changes without writing. Default: `false`

**Workflow**:

1. Validate all records in the directory:
   ```bash
   rootline validate --all <directory> -o json
   ```

2. If errors exist, preview fixes:
   ```bash
   rootline fix --all <directory> --dry-run -o json
   ```

3. Review the proposed fixes and apply when ready:
   ```bash
   rootline fix --all <directory>
   ```

4. Verify the fixes resolved the issues:
   ```bash
   rootline validate --all <directory> -o json
   ```

**When to use**: After modifying records, before committing changes, or when validation reports errors you want to resolve automatically (e.g., enum corrections, missing required fields).

**Rootline tool**: `validate`, `fix`
