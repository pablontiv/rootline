---
estado: In Progress
tipo: task
---
# T004: Design read-only versus mutating tool activation and confirmations.

**Outcome**: [O02 Design Pi extension architecture](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T002-design-rootline-cli-runner.md]]

## Preserva

- INV1: The extension treats Rootline CLI JSON as the integration boundary; it does not import Go internal packages.
  - Verificar: Check architecture docs and implementation tasks.

## Contexto

Esta task forma parte de O02 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Design read-only versus mutating tool activation and confirmations.

## Alcance

**In**:
1. Mutating tools are separated from read-only tools.
2. The design states when user confirmation is required.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T002-design-rootline-cli-runner.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Mutating tools are separated from read-only tools.
- The design states when user confirmation is required.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Pi tool_call hooks`
- `Rootline mutating commands`

## Permission Model Design

### 1. Tool Classification Table

All Rootline CLI tools fall into three permission classes based on their filesystem impact:

| Tool | Class | Purpose | Modifies | Requires Confirmation |
|------|-------|---------|----------|----------------------|
| `query` | read-only | Search and filter records by frontmatter fields | Nothing | No |
| `describe` | read-only | Introspect schema and rules for a directory | Nothing | No |
| `validate` | read-only | Validate a single file against schema rules | Nothing | No |
| `validate --all` | read-only | Batch validate all records in a tree | Nothing | No |
| `tree` | read-only | Display directory hierarchy with metadata | Nothing | No |
| `stats` | read-only | Aggregate statistics across records | Nothing | No |
| `graph` | read-only | Extract dependency graph from wiki-links | Nothing | No |
| `explain` | read-only | Trace field origins (schema, derivation, frontmatter) | Nothing | No |
| `analyze` | read-only | Infer schema gaps and governance issues | Nothing | No |
| `fix --dry-run` | read-only | Preview auto-repair changes without applying | Nothing | No |
| `fix --all` | data-mutating | Auto-repair validation errors in frontmatter | Markdown frontmatter | Yes (after --dry-run) |
| `set` | data-mutating | Set frontmatter fields or document sections | Markdown frontmatter/body | Yes (after --dry-run) |
| `repair apply` | data-mutating | Apply data-only repair proposals from analyze report | Markdown frontmatter | Yes (after --dry-run) |
| `schema apply` | schema-mutating | Apply schema-modifying inferences to `.stem` files | `.stem` files | Yes (explicit approval) |
| `new` | schema-mutating | Create new `.stem` schema files | `.stem` files | Yes |
| `migrate` | schema-mutating | Migrate schema across multiple `.stem` files | `.stem` files | Yes (after preview) |

**Legend**:
- **read-only**: Executes immediately, no confirmation needed. Safe for agent invocation.
- **data-mutating**: Modifies Markdown document frontmatter or body sections. Requires `--dry-run` preview and explicit user confirmation before actual mutation.
- **schema-mutating**: Modifies `.stem` schema files. Requires explicit, typed user confirmation (e.g., user types "apply" or similar affirmation).

### 2. Activation Rules

The Pi runner enforces permission rules per class:

#### Read-Only Tools
- **Activation**: Execute immediately without prompting.
- **Result**: Return JSON directly to the model.
- **Confirmation field**: None.
- **Use case**: Agents can invoke freely for analysis, discovery, and validation.

**Example**:
```json
{
  "tool": "rootline-query",
  "params": { "path": "docs/roadmap", "filter": "estado==\"In Progress\"" }
}
// → Executes immediately, returns results
```

#### Data-Mutating Tools
- **First invocation**: Runner MUST inject `--dry-run` if not already present.
- **Runner response**: Include `"confirmation_required": true` field and display the diff in human-readable form.
- **User action**: User explicitly confirms ("apply", "yes", "OK", etc.) or rejects the proposed changes.
- **Second invocation**: On confirmation, runner executes the same command without `--dry-run`.
- **Result**: Return the actual mutation result (files modified, fields changed).

**Runner behavior**:
```
1. Agent calls fix --all (without --dry-run)
2. Runner injects --dry-run → fix --all --dry-run
3. Runner returns diff + confirmation_required: true
4. User confirms
5. Runner executes fix --all (no --dry-run)
6. Return actual mutation result
```

**Example flow**:
```json
// Step 1: Agent submits without --dry-run
{
  "tool": "rootline-fix",
  "params": { "path": "docs/roadmap", "all": true }
}

// Step 2: Runner returns dry-run preview
{
  "confirmation_required": true,
  "preview": {
    "path": "docs/roadmap/...",
    "changes": [
      { "file": "T001.md", "field": "estado", "before": "Invalid", "after": "In Progress" },
      { "file": "T002.md", "field": "status", "before": "", "after": "Completed" }
    ]
  },
  "message": "Review changes above. Type 'apply' to proceed."
}

// Step 3: User types 'apply'
// Step 4: Runner executes fix --all (without --dry-run)
// Step 5: Return actual mutation result
{
  "version": 1,
  "kind": "rootline/fix-batch",
  "results": [ ... ],
  "summary": { "total": 2, "fixed": 2, "skipped": 0 }
}
```

#### Schema-Mutating Tools
- **Activation**: Runner shows a `--dry-run` preview diff (if supported by the tool).
- **Confirmation type**: Explicit typed approval — user must type a confirmation phrase like "apply <signature>" or similar.
- **Second invocation**: On explicit approval, runner executes without `--dry-run`.
- **Result**: Return the actual schema mutation result.

**Runner behavior**:
```
1. Agent calls schema apply
2. Runner may execute with --dry-run first to show diff (tool-dependent)
3. Runner returns diff + explicit confirmation request
4. User types "apply <sig>" or similar affirmation
5. Runner executes schema apply (actual mutation)
6. Return schema mutation result
```

**Example flow**:
```json
// Step 1: Agent calls schema apply (reads report from stdin)
{
  "tool": "rootline-schema-apply",
  "params": { "report": "analyze_output.json" }
}

// Step 2: Runner returns diff
{
  "confirmation_required": true,
  "confirmation_type": "explicit",
  "diff": {
    "file": "docs/roadmap/.stem",
    "changes": [
      { "operation": "extend_enum", "field": "estado", "values": ["On Hold"] },
      { "operation": "add_aggregate", "field": "completion_rate", "formula": "..." }
    ]
  },
  "message": "Schema changes require explicit approval. Type 'apply <sig>' to proceed."
}

// Step 3: User types 'apply <sig>'
// Step 4: Runner executes schema apply (actual)
// Step 5: Return result
{
  "version": 1,
  "kind": "rootline/apply",
  "stem_path": "docs/roadmap/.stem",
  "modified": true,
  "changes": [ ... ]
}
```

### 3. Dry-Run Enforcement

The Pi runner enforces dry-run discipline for mutating tools:

#### Data-Mutating Tools
- If the agent submits `fix --all` or `set` **without** `--dry-run`, the runner:
  1. Injects `--dry-run` automatically
  2. Executes the command with `--dry-run`
  3. Returns the preview result with `confirmation_required: true`
  4. Waits for explicit user confirmation
  5. Only after user confirms, re-executes without `--dry-run`

#### Schema-Mutating Tools
- If the agent submits `schema apply` or `migrate`, the runner:
  1. Executes with `--dry-run` (if the tool supports it) to show the diff
  2. Returns the diff with explicit confirmation request
  3. Waits for typed user approval (e.g., "apply schema-v1")
  4. Only after explicit approval, executes without `--dry-run`

#### No Bypassing
- The runner NEVER executes a mutating tool without first showing a preview to the user.
- The runner NEVER accepts abbreviated confirmations (e.g., "y", "yes") for schema mutations; explicit typed approval is required.

### 4. Rollback Policy

#### Data-Mutating Operations (`fix`, `set`, `repair apply`)
- **Atomic guarantee**: Each document mutation is atomic (single file rewrite).
- **Partial failure**: If a batch operation (e.g., `fix --all`) fails mid-way, files already modified are NOT rolled back.
- **Recovery**: User must:
  1. Re-run the same tool on the affected directory to fix remaining records.
  2. Or manually revert failed files from version control.
- **Recommendation**: Use `--dry-run` preview to identify problematic records before full execution.

#### Schema-Mutating Operations (`schema apply`, `new`, `migrate`)
- **Atomic guarantee**: `schema apply` modifies a single `.stem` file atomically.
- **Partial failure**: If `migrate` fails mid-way when splitting schemas:
  - Already-created `.stem` files are NOT rolled back.
  - The operation stops and reports which files were modified.
- **Recovery**: Manual `.stem` file cleanup or version control revert.
- **Pre-flight validation**: `migrate` should validate the full migration plan before executing any mutations. If validation fails, zero files are modified (true atomic behavior).

#### Best-Effort vs. Atomic
- **Atomic**: `schema apply` (single file), `set` (single file), `fix` on a single file
- **Best-effort**: `fix --all`, `set --all`, `migrate` (batch operations)

### 5. Context-Free vs. Context-Required Tools

#### Tools Requiring Project Root (Rootline Directory Context)
These tools MUST execute with `cwd` set to a directory containing `.git` or `.stem` ancestors:
- `query`, `validate`, `validate --all`, `tree`, `stats`, `graph`, `explain`
- `describe` (resolves `.stem` parents)
- `analyze`, `apply`, `repair apply`, `fix`, `fix --all`
- `set`
- `schema apply`, `migrate`

**Runner behavior**: Walk up from current directory to find `.git` root. If not found, return error `PROJECT_ROOT_NOT_FOUND`.

#### Tools Context-Free
These tools can execute from any directory without project context:
- `new` (with explicit `--path` parameter to specify output location)
- `init --template` (fetches remote template, does not depend on existing project structure)

**Runner behavior**: For `new` and `init`, allow execution from any directory. Validate `--path` argument separately if provided.

### 6. Confirmation Flow in Pi UI

The Pi runner maintains a **confirmation state machine** for each mutating tool invocation:

```
Initial State
    ↓
[Data-Mutating]
    ↓
Auto-inject --dry-run
    ↓
Execute with --dry-run
    ↓
Return preview + confirmation_required: true
    ↓
User confirms ("apply", "yes", "OK") OR rejects
    ↓
(reject) → Abort, no files modified
(confirm) → Execute without --dry-run
    ↓
Return actual mutation result

---

Initial State
    ↓
[Schema-Mutating]
    ↓
Show --dry-run preview (if supported)
    ↓
Return diff + explicit_confirmation_required: true
    ↓
User types "apply <signature>" OR aborts
    ↓
(abort) → Abort, no .stem files modified
(explicit-confirm) → Execute schema mutation
    ↓
Return schema mutation result
```

### 7. Error Handling and Validation

#### Pre-flight Validation
Before executing any tool, the runner validates:
- Binary availability (`rootline --version` succeeds)
- Project root exists (`.git` directory found) — unless tool is context-free
- Arguments are syntactically valid (no injection attacks)
- Tool supports the requested flags (e.g., `--dry-run` for mutating tools)

#### Validation Failures
- Return `RootlineError` with code `INVALID_ARGUMENTS`
- Do NOT attempt command execution
- Provide suggestions for correction

#### Runtime Failures
- Capture stderr and return in error response
- Include exit code and error code (`ROOTLINE_ERROR`, `TIMEOUT`, `ABORT`)
- For data-mutating commands: If dry-run succeeded but actual execution failed, user can review the diff and re-attempt or abort
