# Inspect and Understand Schema Requirements

**Description**: Examine the schema governing a directory or file to understand required fields, field types, enums, and validation rules.

**Arguments**:
- `path`: File or directory path to inspect (e.g., `docs/epics/E01-feature.md`, `docs/roadmap/`)
- `detail-level` (optional): Show detailed field descriptions and rules. Default: `false`

**Workflow**:

1. Describe the schema for a path:
   ```bash
   rootline describe <path> -o json
   ```

   Returns: field names, types, required status, enum values, validation rules, and `.stem` source.

2. If you need detailed rules and explanations, use explain:
   ```bash
   rootline explain <path> -o json
   ```

   Shows: field origins across `.stem` chain, provenance (which file defined each field), and layered schema.

3. For directory-level schema organization:
   ```bash
   rootline tree <path> -o table
   ```

   Displays hierarchy with metadata, showing how fields are structured across index files and child records.

4. Filter by semantic domain (for complex schemas):
   ```bash
   rootline describe <path> --by-domain -o json
   ```

   Groups fields by semantic domain (e.g., identity, governance, workflow).

**When to use**: Understanding what fields are required before creating a document, validating that a file conforms to schema, learning field types and enum values, or understanding the source of schema requirements across the `.stem` hierarchy.

**Rootline tools**: `describe`, `explain`, `tree`
