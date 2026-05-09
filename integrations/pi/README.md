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

You should see: `rootline-query`, `rootline-describe`, `rootline-validate`, `rootline-tree`, `rootline-stats`, `rootline-new`, `rootline-set`, `rootline-doctor`, `rootline-context`, `rootline-fix`, `rootline-migrate`, `rootline-analyze`, `rootline-apply`.

## Tools

The Pi package provides 9 tools for querying, validating, analyzing, and safely mutating Rootline-governed directories.

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

## Mutation Tools

The Pi package provides safe mutation tools with integrated validation and confirmation guardrails.

### Simple Mutations (O06)

Single-record safe mutations with confirmation:

### rootline-new

Create a new governed record file with frontmatter scaffolded from the effective `.stem` schema.

**When to use:** Use `rootline-new` when you need to create a new record in a directory with `.stem` schema governance. The tool guarantees that the file is scaffolded with correct frontmatter fields and any default values defined in the schema.

**Parameters:**
- `path` (required): Target file path to create
- `force` (optional): Overwrite existing file (default: false)
- `dry_run` (optional): Preview generated content without writing (default: false)
- `confirmed` (optional): Explicit confirmation for mutations in non-interactive mode (default: false)
- `interactive` (optional): Whether Pi is running interactively (default: false)

**Guardrails:**
- Path safety: rejects paths outside project root or with traversal attempts
- Confirmation: requires `confirmed: true` in non-interactive mode
- Post-validation: automatically validates the created file against `.stem` schema

**Example:**
```javascript
// Preview file creation
await tool("rootline-new", {
  path: "docs/roadmap/E14-new-feature/README.md",
  dry_run: true
})

// Create file (interactive mode)
await tool("rootline-new", {
  path: "docs/roadmap/E14-new-feature/README.md",
  interactive: true
})

// Create file (non-interactive, requires confirmation)
await tool("rootline-new", {
  path: "docs/roadmap/E14-new-feature/README.md",
  confirmed: true
})
```

Returns: `created` (boolean), `path`, `validation` (post-write validation details), `content_preview` (for dry-run).

### rootline-set

Update specific frontmatter fields in an existing governed record with automatic post-mutation validation.

**When to use:** Use `rootline-set` when you need to update frontmatter fields in an existing record that is governed by a `.stem` schema. The tool guarantees that the mutation is validated immediately after writing, catching any constraint violations from the schema.

**Parameters:**
- `path` (required): Target file to update
- `fields` (required): Array of field assignments: `[{field: "fieldname", value: "value"}, ...]`
- `dry_run` (optional): Preview changes without writing (default: false)
- `confirmed` (optional): Explicit confirmation for mutations in non-interactive mode (default: false)
- `interactive` (optional): Whether Pi is running interactively (default: false)

**Guardrails:**
- Path safety: rejects paths outside project root or with traversal attempts
- Confirmation: requires `confirmed: true` in non-interactive mode
- Post-validation: automatically validates the updated file against `.stem` schema

**Example:**
```javascript
// Update estado field (interactive mode)
await tool("rootline-set", {
  path: "docs/roadmap/O06-add-safe-mutation-tools/T005-document-safe-mutation-workflows.md",
  fields: [{ field: "estado", value: "Completed" }],
  interactive: true
})

// Preview changes
await tool("rootline-set", {
  path: "docs/roadmap/O06-add-safe-mutation-tools/T005-document-safe-mutation-workflows.md",
  fields: [{ field: "estado", value: "Completed" }],
  dry_run: true
})

// Non-interactive mutation with explicit confirmation
await tool("rootline-set", {
  path: "docs/roadmap/O06-add-safe-mutation-tools/T005-document-safe-mutation-workflows.md",
  fields: [
    { field: "estado", value: "Completed" },
    { field: "tipo", value: "epic" }
  ],
  confirmed: true
})
```

Returns: `updated` (boolean), `path`, `fields_set`, `dry_run_preview` (for dry-run), `validation` (post-write result).

### Complex Bulk Mutations (O07)

Bulk operations that affect multiple files or schema state. These tools require careful use with explicit confirmation and preview workflows. See [Complex Operations](#complex-operations) section for risk model and rollback guidance.

#### rootline-fix

Bulk correction of validation errors across multiple documents (field standardization, enum fixes, required field population).

**Parameters:**
- `path` (required): Directory or file path to analyze and fix
- `dry_run` (optional): Preview changes without writing (default: true)
- `confirmed` (optional): Explicit confirmation for mutations in non-interactive mode (default: false)
- `interactive` (optional): Whether Pi is running interactively (default: false)
- `where` (optional): Filter expression to target specific records (e.g., `estado == 'Pending'`)

**Example:**
```javascript
// Preview fixes
await tool("rootline-fix", {
  path: "docs/roadmap/",
  dry_run: true
})

// Apply fixes (interactive mode)
await tool("rootline-fix", {
  path: "docs/roadmap/",
  dry_run: false,
  interactive: true
})

// Apply fixes with explicit confirmation (non-interactive)
await tool("rootline-fix", {
  path: "docs/roadmap/",
  dry_run: false,
  confirmed: true
})
```

Returns: `fixed` (boolean), `path`, `proposals` (list of corrections), `validation` (post-fix validation).

#### rootline-migrate

Bulk migration of records and schema across a directory hierarchy (field renames, type changes, hierarchical restructuring).

**Parameters:**
- `path` (required): Directory path to migrate
- `dry_run` (optional): Preview changes without writing (default: true)
- `confirmed` (optional): Explicit confirmation for mutations in non-interactive mode (default: false)
- `interactive` (optional): Whether Pi is running interactively (default: false)

**Example:**
```javascript
// Preview migration plan
await tool("rootline-migrate", {
  path: "docs/roadmap/",
  dry_run: true
})

// Apply migration (interactive mode)
await tool("rootline-migrate", {
  path: "docs/roadmap/",
  dry_run: false,
  interactive: true
})
```

Returns: `migrated` (boolean), `path`, `plan` (migration steps), `validation` (post-migration validation).

#### rootline-analyze

Analyze records to detect schema inference opportunities, governance gaps, and validation improvements (read-only, no confirmation required).

**Parameters:**
- `path` (required): Directory path to analyze
- `incremental` (optional): Skip patterns already covered by .stem files (default: false)
- `threshold` (optional): Confidence threshold for inferences (0-100, default: 80)

**Example:**
```javascript
// Generate analysis report
await tool("rootline-analyze", {
  path: "docs/roadmap/",
  incremental: true
})

// Save report to file for review and apply later
const report = await tool("rootline-analyze", {
  path: "docs/roadmap/"
})
// Then save to disk: fs.writeFileSync("analyze-report.json", JSON.stringify(report))
```

Returns: `kind` ("rootline/analyze-report"), `version` (1), `directory`, `timestamp`, `summary`, `inferences` (list of detected patterns), `governance_gaps`, `validation_improvements`.

#### rootline-apply

Apply proposals from an analysis report (schema changes to .stem files, or data repairs to documents).

**Parameters:**
- `report_path` (required): Path to a rootline-analyze report (JSON file)
- `mode` (optional): "schema", "repair", or "both" (default: "both")
- `dry_run` (optional): Preview changes without writing (default: true)
- `confirmed` (optional): Explicit confirmation for mutations in non-interactive mode (default: false)
- `interactive` (optional): Whether Pi is running interactively (default: false)

**Example:**
```javascript
// Preview proposals from a report
await tool("rootline-apply", {
  report_path: "analyze-report.json",
  mode: "both",
  dry_run: true
})

// Apply schema changes only
await tool("rootline-apply", {
  report_path: "analyze-report.json",
  mode: "schema",
  dry_run: false,
  interactive: true
})

// Apply data repairs only (non-interactive with confirmation)
await tool("rootline-apply", {
  report_path: "analyze-report.json",
  mode: "repair",
  dry_run: false,
  confirmed: true
})
```

Returns: `applied` (boolean), `dry_run`, `mode`, `schema_result?` (if mode includes schema), `repair_result?` (if mode includes repair), `validation` (post-apply validation).

## When to Use Direct Edit/Write vs. Mutation Tools

| Scenario | Use |
|----------|-----|
| Creating a new governed record (file with `.stem` schema in parent dir) | `rootline-new` |
| Updating frontmatter fields in a governed record | `rootline-set` |
| Ungoverndered file (no `.stem` schema) | Direct write/edit tools |
| Modifying document body only (no frontmatter changes) | Direct write/edit tools |
| Structural changes (reorganizing sections) | Direct write/edit tools, then run `rootline validate` |

## Complex Operations

O07 exposes complex bulk operations that can affect multiple files or schema state. These operations have integrated guardrails to prevent accidental mutations and provide clear rollback guidance.

### When Operations Require Confirmation

All mutation operations default to **safe-by-default** behavior:

| Operation | Default Mode | To Actually Apply |
|-----------|--------------|-------------------|
| `rootline-fix` | `dry_run: true` | Set `dry_run: false` + `confirmed: true` |
| `rootline-migrate` | `dry_run: true` | Set `dry_run: false` + `confirmed: true` |
| `rootline-apply` (repair) | `dry_run: true` | Set `dry_run: false` + `confirmed: true` |
| `rootline-apply` (schema) | `dry_run: true` | Set `dry_run: false` + `confirmed: true` |
| `rootline-analyze` | N/A | Read-only operation |

**Interactive vs. Non-Interactive Behavior:**
- **Interactive mode** (`interactive: true`): Agent prompts the user before applying mutations
- **Non-interactive/headless mode**: Operation is blocked unless `confirmed: true` is explicitly set

This prevents silent mutations and ensures explicit user intent.

### Risk Model

| Operation | Risk Level | Affects | Scope | Rollback |
|-----------|-----------|---------|-------|----------|
| `rootline-analyze` | None | None | Read-only | N/A |
| `rootline-fix` | Medium | Document frontmatter | Single or bulk documents | `git reset` or `git checkout` |
| `rootline-apply (repair)` | Medium | Document frontmatter | Single or bulk documents | `git reset` or `git checkout` |
| `rootline-migrate` | High | Documents + .stem files | Multiple files in hierarchy | `git reset --hard` or `git restore` |
| `rootline-apply (schema)` | High | .stem schema files | Schema definitions | `git restore` or manual `.stem` revert |

**Risk Factors:**
- **None**: Read-only analysis, no mutations
- **Medium**: Frontmatter changes only, documents remain valid, easy to revert
- **High**: Schema changes or bulk document mutations, requires careful review of diffs

### Git Checkpoint and Rollback Procedure

**Before any complex operation**, create a checkpoint:

```bash
git add -A
git commit -m "checkpoint before rootline [operation]"
# Example: "checkpoint before rootline migrate docs/roadmap/"
```

**If something goes wrong:**

**Option 1: Revert entire operation**
```bash
# Undo to the checkpoint commit
git reset --hard HEAD~1
```

**Option 2: Revert only changed documents (for frontmatter issues)**
```bash
# Restore specific files
git checkout -- docs/roadmap/O07-*/
```

**Option 3: Revert only schema files**
```bash
# If only .stem files were changed
git restore docs/roadmap/.stem
```

**Verification after rollback:**
```bash
# Verify documents are back to checkpoint
rootline validate --all docs/roadmap/ --output json

# Check git status
git status
```

### Complex Operations Workflow

**Recommended workflow for bulk operations:**

1. **Generate and preview**
   ```javascript
   // Step 1: Create checkpoint
   // Run: git add -A && git commit -m "checkpoint before rootline analyze"
   
   // Step 2: Analyze or preview (dry-run)
   await tool("rootline-analyze", {
     path: "docs/roadmap/",
     incremental: true
   })
   
   // Or preview fixes
   await tool("rootline-fix", {
     path: "docs/roadmap/",
     dry_run: true
   })
   ```

2. **Review and approve**
   - Inspect the analysis report or dry-run preview
   - Look for unexpected changes or breaking modifications
   - Decide whether to proceed or adjust strategy

3. **Apply with confirmation**
   ```javascript
   // Step 3: Apply changes (in interactive mode, user is prompted)
   await tool("rootline-fix", {
     path: "docs/roadmap/",
     dry_run: false,
     confirmed: true,
     interactive: true
   })
   ```

4. **Validate and commit**
   ```javascript
   // Step 4: Validate results
   await tool("rootline-validate", {
     path: "docs/roadmap/"
   })
   
   // If validation passes: git add && git commit
   // If validation fails: git reset to checkpoint
   ```

### When NOT to Use Complex Operations

- **Uncontrolled bulk changes**: If unsure about the scope, start with a filtered query to understand the impact first
- **Critical schema changes**: For schema migrations affecting the entire hierarchy, use a feature branch and test on a copy first
- **Production-like data**: Always test migrations on a smaller subset before running on the full dataset

## Bulk Operations and Non-Goals

This package exposes single-record mutations with integrated validation, and complex bulk operations with guardrails. **Single-record operations** (O06) are always safe:

| Operation | Scope | Guardrails |
|-----------|-------|-----------|
| `rootline-new` | Single file creation | Confirmation required in non-interactive mode |
| `rootline-set` | Single field update | Confirmation required + post-validation |

**Complex bulk operations** (O07) require explicit confirmation and preview workflows:
- `rootline-fix` (bulk field correction across multiple records)
- `rootline-migrate` (schema migration and bulk changes)
- `rootline-analyze` (inference and analysis)
- `rootline-apply` (repair or schema application from proposals)

**Not in scope:**
- Batch record creation — use `rootline-new` for single records, or `rootline-migrate` to bulk-restructure existing records
- Automated rollback — manual git-based rollback is required; see [Git checkpoint/rollback](#git-checkpoint-and-rollback-procedure) above

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

### Smoke Tests

Headless Pi smoke tests verify package discovery and core workflows without requiring an interactive Pi session:

```bash
# Run all smoke tests
bash integrations/pi/smoke-tests/test-discovery.sh
bash integrations/pi/smoke-tests/test-query.sh
bash integrations/pi/smoke-tests/test-validate.sh
```

**Test descriptions:**

- `test-discovery.sh`: Verifies that Rootline tools are discoverable (doctor, query, validate, tree, stats, describe) and that Pi extension infrastructure is in place
- `test-query.sh`: Runs a read-only query on the roadmap and verifies results (counts index files)
- `test-validate.sh`: Runs `rootline validate --all docs/roadmap/` and verifies exit code 0

**When smoke tests run:**
- Locally: Run manually with `bash integrations/pi/smoke-tests/test-*.sh`
- In CI: Runs after `bun test` in the `pi-extensions` job if Pi is installed (gracefully skipped if not)

### Acceptance Criteria Status

1. **Smoke tests verify tools/commands are discoverable**: ✓ 
   - `test-discovery.sh` checks for 6 core tool implementations
   - Package manifest and runner infrastructure verified
   - Verified: 2026-05-09

2. **At least one read-only Rootline query succeeds through Pi**: ✓ 
   - `test-query.sh` runs `rootline query` headless and returns valid JSON
   - Queries the roadmap and counts index files
   - Verified: 2026-05-09

3. **`rootline validate --all docs/roadmap/` returns exit 0**: ✓ 
   - `test-validate.sh` runs validation
   - Command executes successfully with exit code 0
   - All records validate with 0 errors
   - Verified: 2026-05-09

## Release Strategy

### Package Identity

- **Package name**: `pi-rootline` (defined in `package.json`)
- **Current location**: Co-located in `pablontiv/rootline` repository under `integrations/pi/`
- **Separate npm package**: Deferred — no separate npm distribution needed at this stage. Package is installed via Git or local path.

### Installation Methods

#### Local Development (Current)

For development and local testing, install directly from a filesystem path:

```bash
# From within rootline repo
pi install -l ./integrations/pi

# From anywhere, using absolute path
pi install -l /home/shared/rootline/integrations/pi
```

This method is recommended for:
- Local development workflows
- Integration testing with your own Pi instance
- Testing the install flow in isolated environments

#### Git Install (Future)

When Pi's package ecosystem matures, users will be able to install via Git reference:

```bash
# Proposed syntax (pending Pi ecosystem maturity)
pi install github:pablontiv/rootline/integrations/pi
# or similar Git-based installation syntax
```

Currently not supported by Pi's install system — this is a deferred feature pending ecosystem stabilization.

### Versioning Strategy

**Version inheritance**: The `pi-rootline` package version is **not** independently versioned. Instead:

- The `package.json` version field (`0.1.0`) is a placeholder and does not drive releases
- Actual package compatibility is tied to the rootline binary version
- When rootline releases a new version (e.g., `v0.9.100` → `v0.10.0`), the pi-rootline package is included as part of that release
- No separate semver bump or npm package release occurs — the source code is packaged as part of rootline's GitHub Release artifacts

**Rationale**: Since the package is co-located in the rootline repo and tightly coupled to the rootline binary's functionality, independent versioning would create confusion about compatibility. Users install `pi-rootline` to extend a specific rootline version, so tying them together simplifies dependency management.

### Compatibility

#### Peer Dependencies

- **`@earendil-works/pi-coding-agent: "*"`** — Pi coding agent package (installed as a dependency of Pi itself)
- **`rootline` binary ≥ current version** — The pi-rootline package requires the rootline CLI binary to be available in `$PATH`

#### Verification

To verify compatibility before use, run:

```bash
# Check Rootline binary is installed
which rootline
rootline --version

# Check Pi package is installed
pi list | grep rootline

# Run health check
pi tools | grep rootline-
# Should show: rootline-query, rootline-describe, rootline-validate, rootline-tree, rootline-stats, rootline-doctor, rootline-context, rootline-fix, rootline-migrate, rootline-analyze, rootline-apply
```

If `rootline` is not found, see [Troubleshooting > rootline: command not found](#rootline-command-not-found).

### Location Decision

**Stays in `pablontiv/rootline`**: The Pi extension package remains in this repository. Benefits:

- **Single source of truth**: Rootline CLI + Pi extension stay synchronized — no async release issues
- **Unified testing**: CI runs both CLI and Pi extension tests in the same workflow
- **Simplified dependency management**: Users install one repository for both CLI and extensions
- **Deferred extraction**: If the Pi ecosystem matures (separate distribution, npm publishing, etc.), extraction to a separate repo can be considered in a future release without breaking existing workflows

### Future Evolution

If circumstances change (e.g., Pi package ecosystem becomes a primary distribution channel, separate team ownership, etc.), the package **may** be extracted to:

- A separate `pi-rootline` npm package on npm registry
- A dedicated `pablontiv/pi-rootline` GitHub repository
- A community marketplace or plugin distribution system

Any extraction would:
1. Maintain backward compatibility with local path installs during transition
2. Update this README with new installation instructions
3. Require a major version bump (v1.0+) to signal the structural change
4. Be coordinated with the main rootline release cycle

Until then, the co-located approach remains the canonical distribution method.

### Integration Test Target

The `/home/shared/pinata/` repository (`pi-pinata` personal extension bundle) can be used as a real-world integration test target:

```bash
# Install pi-rootline into your personal Pi instance
pi install -l /home/shared/rootline/integrations/pi

# Verify tools are available in Pi
pi tools | grep rootline-

# Test a query
pi list | grep -c rootline
# Should show at least one match: "pi-rootline" or similar
```

This validates the end-to-end install flow without requiring mocking or artificial test environments.

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
