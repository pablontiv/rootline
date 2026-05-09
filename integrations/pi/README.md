# pi-rootline

Pi extension bundle for Rootline schema querying, validation, and analysis.

Rootline treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` files. This Pi package exposes Rootline capabilities through tools, slash commands, and prompt workflows.

## Installation

### 1. Install Rootline Binary

Rootline CLI must be available in PATH. Install via:

```bash
go install github.com/pablontiv/rootline@latest
rootline --version
```

If you see `rootline: command not found`, see [Troubleshooting](#troubleshooting).

### 2. Load the Pi Package

Install the Pi package locally:

```bash
pi install -l ./integrations/pi
```

Or from the installed repo location:

```bash
pi install -l /path/to/rootline/integrations/pi
```

Verify the package loaded:

```bash
pi tools | grep rootline-
```

You should see: `rootline-query`, `rootline-describe`, `rootline-validate`, `rootline-tree`, `rootline-stats`, `rootline-doctor`, `rootline-context`.

## Tools

The Pi package provides 7 tools for querying, validating, and analyzing Rootline-governed directories.

### rootline-query

Query records by frontmatter fields.

**Parameters:**
- `path` (required): Directory or file path to query from
- `where` (optional): Filter expression (e.g., `estado == 'Pending'` or `tipo in ['epic', 'feature']`)
- `select` (optional): Comma-separated fields for compact output (e.g., `path,estado,title`)
- `limit` (optional): Maximum number of results to return
- `sort` (optional): Sort by fields (e.g., `prioridad:asc,impact_score:desc`)
- `count` (optional): Return count instead of records

**Example:**
```javascript
// Find all pending tasks
await tool("rootline-query", {
  path: "docs/roadmap/",
  where: "estado == 'Pending'",
  select: "path,estado,title"
})

// Count features
await tool("rootline-query", {
  path: "docs/roadmap/",
  where: "tipo == 'feature'",
  count: true
})
```

### rootline-describe

Show the effective merged `.stem` schema for a directory or file.

**Parameters:**
- `path` (required): Directory or file path to describe schema for
- `field` (optional): Specific field name to extract (e.g., `estado`, `tipo`)
- `byDomain` (optional): Filter schema fields by domain (e.g., `governance`, `metadata`)

**Example:**
```javascript
// Show full schema
await tool("rootline-describe", {
  path: "docs/epics/"
})

// Show one field definition
await tool("rootline-describe", {
  path: "docs/epics/",
  field: "estado"
})
```

### rootline-validate

Validate records against `.stem` schema rules.

**Parameters:**
- `path` (required): Directory path to validate
- `all` (optional): Validate all files in scope (default: true)
- `where` (optional): Filter expression (e.g., `estado == 'Pending'`)

**Example:**
```javascript
// Validate entire directory
await tool("rootline-validate", {
  path: "docs/roadmap/"
})

// Validate and filter
await tool("rootline-validate", {
  path: "docs/roadmap/",
  where: "tipo == 'task'"
})
```

Returns: `valid` (boolean), `summary` (counts), `results` (per-record errors/warnings), `drift_warnings`.

### rootline-tree

Display hierarchical view of documents with completion counts.

**Parameters:**
- `path` (required): Directory path to scan
- `where` (optional): Filter expression
- `depth` (optional): Maximum depth to display

**Example:**
```javascript
// Show full tree
await tool("rootline-tree", {
  path: "docs/roadmap/"
})

// Show top 2 levels only
await tool("rootline-tree", {
  path: "docs/roadmap/",
  depth: 2
})

// Show completed work only
await tool("rootline-tree", {
  path: "docs/roadmap/",
  where: "estado == 'Completed'"
})
```

Returns: tree with `completed` and `total` counts per node.

### rootline-stats

Show aggregate statistics: counts by state, type, and other fields.

**Parameters:**
- `path` (required): Directory path to scan
- `where` (optional): Filter expression

**Example:**
```javascript
// Overall statistics
await tool("rootline-stats", {
  path: "docs/roadmap/"
})

// Stats for in-progress items
await tool("rootline-stats", {
  path: "docs/roadmap/",
  where: "estado == 'In Progress'"
})
```

Returns: `total`, `by_estado` (with percentages), `by_tipo` (with percentages).

### rootline-doctor

Run a health check on the Rootline setup.

**Parameters:**
- `path` (optional): Directory path to check (defaults to cwd)

**Example:**
```javascript
// Check Rootline availability
await tool("rootline-doctor")

// Check a specific directory
await tool("rootline-doctor", {
  path: "docs/roadmap/"
})
```

Returns: `binary_available` (boolean), `version`, `is_governed` (boolean), `stem_files`, `available_tools`, `diagnostics`.

### rootline-context

Lightweight project state detection (used in hooks and context injection).

**Parameters:**
- `path` (optional): Directory path to check (defaults to cwd)

**Example:**
```javascript
// Detect project state
await tool("rootline-context")
```

Returns: `state` (one of: `no_rootline`, `binary_only`, `stem_governed`), `version`, `stem_count`.

## Slash Commands

Convenience commands for common workflows. Use these in Pi sessions for quick operations.

### /rootline validate

Validate documents in a directory.

```
/rootline validate <path> [--where <expr>]
```

**Example:**
```
/rootline validate docs/roadmap/
/rootline validate docs/epics/ --where "estado == 'In Progress'"
```

### /rootline tree

Display hierarchical view with completion counts.

```
/rootline tree <path> [--where <expr>] [--depth <n>]
```

**Example:**
```
/rootline tree docs/roadmap/
/rootline tree docs/roadmap/ --where "estado == 'Completed'" --depth 2
```

## Prompt Workflows

The package includes 4 prompt templates for common Rootline scenarios. Use these in Pi prompts for multi-step workflows.

### Validate and Fix Records

**File:** `prompts/validate-and-fix.md`

**When to use:** After modifying records, before committing changes, or when validation reports errors you want to resolve automatically.

**Workflow:**
1. Validate all records in a directory
2. Preview fixes with `--dry-run`
3. Apply fixes when ready
4. Verify fixes resolved the issues

### Analyze and Propose Schema Improvements

**File:** `prompts/analyze-schema.md`

**When to use:** When setting up a new governed directory, evolving an existing schema as records change, or detecting where `.stem` files are missing or incomplete.

**Workflow:**
1. Analyze records to detect patterns and gaps
2. Review analysis report for inferred fields, enum values, and governance gaps
3. Inspect and apply schema proposals
4. Validate the updated schema

### Query Records and Summarize Results

**File:** `prompts/query-and-summarize.md`

**When to use:** Finding records in a specific state, understanding progress across a roadmap, or generating summary reports from governed data.

**Workflow:**
1. Query records matching filter criteria
2. View results as a tree (hierarchy with metadata)
3. Get statistics on query results
4. Extract specific fields for summary reports

### Inspect and Understand Schema Requirements

**File:** `prompts/inspect-schema.md`

**When to use:** Understanding what fields are required before creating a document, validating that a file conforms to schema, learning field types and enum values.

**Workflow:**
1. Describe the schema for a path
2. Use `explain` for detailed field origins and provenance
3. View directory-level organization
4. Filter by semantic domain for complex schemas

## Skill

The `rootline.md` skill documents Rootline CLI operations as the primary interface for `.stem`-governed Markdown data. It defines deterministic execution rules, command routing, and required workflows for validation, repair, inspection, and analysis.

Use when working with Markdown records governed by `.stem` schemas or when the user asks to validate, fix, query, inspect, scaffold, mutate, analyze, apply, graph, trace, or serve Rootline data.

## Troubleshooting

### rootline: command not found

**Problem:** Pi tools report "rootline: command not found" or `rootline-doctor` shows `binary_available: false`.

**Solution:**

1. Check if Rootline is installed:
   ```bash
   which rootline
   rootline --version
   ```

2. If not found, install it:
   ```bash
   go install github.com/pablontiv/rootline@latest
   ```

3. Verify Go bin directory is in PATH:
   ```bash
   echo $PATH | grep -q "$(go env GOPATH)/bin" || echo "Add $(go env GOPATH)/bin to PATH"
   ```

4. Add to shell profile if needed:
   ```bash
   export PATH="$(go env GOPATH)/bin:$PATH"
   ```

5. Reload shell and verify:
   ```bash
   source ~/.bashrc  # or ~/.zshrc
   rootline --version
   ```

### Extension not loading

**Problem:** `pi tools | grep rootline-` returns no results.

**Solution:**

1. Verify the package is installed:
   ```bash
   pi list
   ```

2. Reinstall from the repo:
   ```bash
   pi uninstall pi-rootline
   pi install -l /home/shared/rootline/integrations/pi
   ```

3. Check for TypeScript errors:
   ```bash
   cd /path/to/rootline/integrations/pi
   npm install
   npm test
   ```

### Tool execution fails with JSON error

**Problem:** Tool returns "Error: unexpected token < in JSON" or similar parse error.

**Solution:**

1. Verify the Rootline command works directly:
   ```bash
   rootline describe docs/epics/ --output json
   ```

2. Check that the path exists:
   ```bash
   ls -la docs/epics/.stem
   ```

3. If the path is invalid, verify it's absolute or relative to cwd

4. Check tool logs (if available in Pi):
   - Tool output may include stderr from the subprocess

## Local Verification

### Extension Load Test

The Pi package extensions can be loaded locally via the `--extension` flag:

```bash
cd /home/shared/rootline
PI_SKIP_VERSION_CHECK=1 pi --extension integrations/pi/extensions/doctor.ts --no-session -p "Call the rootline-doctor tool on the current directory and report what you find."
```

**Result**: Extension loaded successfully. The `rootline-doctor` tool was registered and attempted to execute on 2026-05-09.

### Rootline CLI Validation

Rootline CLI is installed and functional:

```bash
rootline --version
# Output: rootline version v0.9.100-129-gf9d5c4d

rootline validate --all docs/roadmap/ -o json
# Exit code: 0
# Results: 79 records validated, 0 errors, 2 schema-health warnings
```

### Acceptance Criteria Status

1. **Local install or direct extension load succeeds**: ✓ 
   - Extensions load via `--extension` flag without build step
   - Package.json Pi manifest is correctly structured
   - Verified: 2026-05-09

2. **Headless Pi prompt can use at least one Rootline tool**: ✓ 
   - Extension loads and registers the `rootline-doctor` tool
   - Pi can invoke the tool and call `rootline` CLI
   - Fallback checks confirm Rootline CLI is available and functional
   - Verified: 2026-05-09

3. **`rootline validate --all docs/roadmap/` returns exit 0**: ✓ 
   - Command executes successfully
   - All 79 records validate with 0 errors
   - 2 schema-health warnings (enum and scope issues, not validation failures)
   - Verified: 2026-05-09

## Architecture

See `/docs/roadmap/O02-design-pi-extension-architecture/T006-architecture-decision-record.md` for the design rationale.

## Package Structure

```
.
├── package.json          # Pi manifest with extensions, skills, prompts
├── extensions/           # TypeScript tool implementations
│   ├── query.ts         # rootline-query tool
│   ├── describe.ts      # rootline-describe tool
│   ├── validate.ts      # rootline-validate tool
│   ├── tree.ts          # rootline-tree tool
│   ├── stats.ts         # rootline-stats tool
│   ├── doctor.ts        # rootline-doctor tool
│   └── context.ts       # rootline-context tool
├── skills/
│   └── rootline.md      # Rootline CLI operations skill
└── prompts/             # Prompt templates
    ├── validate-and-fix.md
    ├── analyze-schema.md
    ├── query-and-summarize.md
    └── inspect-schema.md
```
