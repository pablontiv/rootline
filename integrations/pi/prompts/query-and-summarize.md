# Query Records and Summarize Results

**Description**: Search for records matching criteria and summarize findings by state, type, or other fields.

**Arguments**:
- `directory`: Path to search (e.g., `docs/epics/`, `docs/roadmap/`)
- `filter` (optional): Expression to filter records (e.g., `estado == 'In Progress'`, `tipo == 'task' && length(tags) > 0`)
- `group-by` (optional): Field to group results by (e.g., `estado`, `tipo`, `domain`)
- `summary-field` (optional): Field to extract for summary (e.g., `summary`, `estado`, `ejecutable_en`)

**Workflow**:

1. Query records matching filter criteria:
   ```bash
   rootline query <directory> --where "<filter>" -o json
   ```

   Example filters:
   - `estado == 'Completed'` — find finished tasks
   - `tipo == 'task' && length(tags) > 0` — find tagged tasks
   - `ejecutable_en == 'agent'` — find tasks executable by agents

2. View results as a tree (hierarchy with metadata):
   ```bash
   rootline tree <directory> --where "<filter>" -o table
   ```

3. Get statistics on query results:
   ```bash
   rootline stats <directory> --where "<filter>" -o json
   ```

   Shows: record count, state distribution, type counts, and other aggregate metrics.

4. For detailed field values, extract specific fields:
   ```bash
   rootline query <directory> --where "<filter>" --field <summary-field> -o json
   ```

**When to use**: Finding records in a specific state, understanding progress across a roadmap, locating tasks of a particular type or assigned to an agent, or generating summary reports from governed data.

**Rootline tools**: `query`, `tree`, `stats`
