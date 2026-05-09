# /rootline tree Command

**Description**: Convenience slash command to display hierarchical view of documents with completion counts.

**Usage**: `/rootline tree <path> [--where <expr>] [--depth <n>]`

**Arguments**:
- `path` (required): Directory path to scan (e.g., `docs/epics/`, `docs/roadmap/`)
- `--where` (optional): Filter expression using expr syntax (e.g., `estado == 'Completed'`)
- `--depth` (optional): Maximum depth to display (numeric value, e.g., `2` for two levels deep)

**Workflow**:

1. Accept the path argument and optional filter/depth parameters from the user.

2. Call the `rootline-tree` tool with the provided parameters:
   ```
   rootline-tree(path: "<path>", where: "<optional-filter>", depth: <optional-depth>)
   ```

3. Display the tree structure with:
   - Hierarchical directory layout
   - Completion counts for each node (completed / total)
   - Directory names and paths
   - Optional filtering by status or type
   - Optional depth limiting for condensed views

4. Provide a summary at the root level showing:
   - Total records in tree
   - Total completed records
   - Overall completion percentage

**When to use**:
- Getting a quick overview of project structure with completion metrics
- Visualizing progress across a hierarchical roadmap
- Finding directories with incomplete work using filters
- Understanding directory organization without reading individual files
- Limiting display depth for high-level views (e.g., `--depth 2` for epic-level overview)

**Example filters**:
- `estado == 'Completed'` — show only completed records
- `tipo == 'feature'` — show only feature-type records
- `estado != 'Completed'` — show pending or in-progress work

**Rootline tool**: `rootline-tree`
