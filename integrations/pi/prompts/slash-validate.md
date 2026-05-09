# /rootline validate Command

**Description**: Convenience slash command to validate documents in a directory against their .stem schema rules.

**Usage**: `/rootline validate <path> [--where <expr>]`

**Arguments**:
- `path` (required): Directory path to validate (e.g., `docs/epics/`, `docs/roadmap/`)
- `--where` (optional): Filter expression using expr syntax (e.g., `estado == 'In Progress'`)

**Workflow**:

1. Accept the path argument and optional filter expression from the user.

2. Call the `rootline-validate` tool with the provided path:
   ```
   rootline-validate(path: "<path>", all: true, where: "<optional-filter>")
   ```

3. Display the validation results:
   - **Valid**: If all records pass validation, report success and summary.
   - **Invalid**: If validation errors exist, show:
     - Total records scanned
     - Number of invalid records
     - Error count and warnings count
     - For each error: path, field name, rule violated, and message
   - **Drift warnings**: If any drift warnings are detected, display them separately.

4. Provide actionable feedback:
   - If errors found, suggest using `/rootline fix-validate` for automatic corrections.
   - If only warnings, explain their severity and when they might require attention.

**When to use**: 
- Quick validation checks on a directory without detailed analysis
- Verifying that records comply with schema rules before committing changes
- Filtering validation to specific records using `--where` expressions
- Debugging validation failures to understand what schema rules are violated

**Rootline tool**: `rootline-validate`

**Exit status**: Returns 0 if all records are valid, 1 if errors exist.
