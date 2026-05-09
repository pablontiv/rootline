# Analyze and Propose Schema Improvements

**Description**: Analyze records in a directory to detect schema patterns, missing definitions, and governance gaps. Generate proposals for schema improvements.

**Arguments**:
- `directory`: Path to the directory containing records to analyze (e.g., `docs/epics/`, `docs/roadmap/`)
- `incremental` (optional): Skip patterns already covered by existing `.stem` files. Default: `false`

**Workflow**:

1. Analyze records and detect patterns:
   ```bash
   rootline analyze <directory> -o json
   ```

   Or with incremental filtering (skip already-covered patterns):
   ```bash
   rootline analyze <directory> --incremental -o json
   ```

2. Review the analysis report. The report includes:
   - **Inferred fields**: new fields detected across records
   - **Enum values**: candidate enum values for fields
   - **Governance gaps**: directories without `.stem` files, schema coverage issues
   - **Structure patterns**: naming conventions, directory hierarchy patterns

3. If schema changes are proposed, inspect and apply:
   ```bash
   rootline schema apply --report <proposals.json> --dry-run
   rootline schema apply --report <proposals.json>
   ```

4. Validate the updated schema:
   ```bash
   rootline validate --all <directory> -o json
   ```

**When to use**: When setting up a new governed directory, evolving an existing schema as records change, or detecting where `.stem` files are missing or incomplete.

**Rootline tools**: `analyze`, `schema apply`, `validate`
