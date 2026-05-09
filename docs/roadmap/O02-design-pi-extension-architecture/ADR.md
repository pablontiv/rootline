---
tipo: adr
decision_date: 2026-05-09
status: Accepted
---

# ADR: Pi Rootline Extension Architecture

## Context

The Pi extension for Rootline must provide safe, maintainable access to Rootline capabilities (schema querying, validation, analysis, schema migration) for agent-driven workflows within Pi. The extension needs to:

1. **Integrate safely**: Avoid Go package imports; use only stable CLI contracts
2. **Maintain boundaries**: Separate read-only, data-mutating, and schema-mutating operations
3. **Be predictable**: Consistent error handling, JSON contracts, timeout management
4. **Support agent workflows**: Enable multi-step reasoning with confirmation checkpoints

The root problem: Rootline's Go internals are under active development and not suitable for long-term embedding. The CLI interface (via JSON output) is stable and versioned, making it the right integration boundary.

## Decision

**Use Rootline CLI JSON output as the sole integration boundary.** The Pi extension will:

- Shell out to the `rootline` binary via subprocess calls (no direct Go imports)
- Consume only JSON version-1 contracts from stdout
- Enforce permission classes (read-only, data-mutating, schema-mutating) via pre-flight validation
- Implement a shared CLI runner handling binary discovery, timeouts, error parsing, and version compatibility
- Define 12 read-only tools and 4 mutating tools with stable parameter/result contracts
- Truncate large outputs and signal `truncated: true` for model-facing responses
- Enforce dry-run discipline for mutating operations

## Package Layout

The Pi Rootline extension lives in a standalone package (e.g., `packages/pi-rootline-extension/` or equivalent in the Pi package ecosystem) with the following structure:

```
pi-rootline-extension/
├── README.md                      # Overview and quick start
├── src/
│   ├── runner.ts                  # Shared CLI runner (binary discovery, timeout, parsing)
│   ├── schemas/
│   │   ├── tools.ts              # Tool parameter and result types (12 read-only + 4 mutating)
│   │   ├── errors.ts             # RootlineError types and error codes
│   │   ├── truncation.ts         # Output truncation limits and render configs
│   │   └── permissions.ts        # Permission class definitions
│   ├── tools/
│   │   ├── read-only/            # Query, describe, validate, tree, stats, graph, explain, analyze, fix --dry-run
│   │   │   ├── query.ts
│   │   │   ├── describe.ts
│   │   │   ├── validate.ts
│   │   │   └── ...
│   │   ├── data-mutating/        # Fix, set, repair apply
│   │   │   ├── fix.ts
│   │   │   ├── set.ts
│   │   │   └── repair-apply.ts
│   │   └── schema-mutating/      # Schema apply, new, migrate
│   │       ├── schema-apply.ts
│   │       ├── new.ts
│   │       └── migrate.ts
│   ├── confirmation.ts            # Confirmation state machine for mutations
│   ├── truncation.ts              # Output truncation logic
│   └── index.ts                   # Public API exports
├── tests/
│   ├── runner.test.ts
│   ├── tools/
│   │   ├── query.test.ts
│   │   └── ...
│   └── integration/               # End-to-end scenarios
└── package.json
```

## Tool Naming Convention

CLI command names are translated to Pi tool names by replacing hyphens and spaces with underscores:

| Rootline CLI Command | Pi Tool Name |
|---|---|
| `rootline query` | `rootline-query` |
| `rootline describe` | `rootline-describe` |
| `rootline validate` | `rootline-validate` |
| `rootline validate --all` | `rootline-validate-all` |
| `rootline tree` | `rootline-tree` |
| `rootline stats` | `rootline-stats` |
| `rootline graph` | `rootline-graph` |
| `rootline explain` | `rootline-explain` |
| `rootline analyze` | `rootline-analyze` |
| `rootline fix --all --dry-run` | `rootline-fix` (read-only wrapper) |
| `rootline fix --all` | `rootline-fix-all` (data-mutating) |
| `rootline set` | `rootline-set` (data-mutating) |
| `rootline repair apply` | `rootline-repair-apply` (data-mutating) |
| `rootline schema apply` | `rootline-schema-apply` (schema-mutating) |
| `rootline new` | `rootline-new` (schema-mutating) |
| `rootline migrate` | `rootline-migrate` (schema-mutating) |

## CLI Boundary

### Subprocess Execution

All tool invocations execute `rootline` as a subprocess with:

1. **Argument passing**: Array-based (not shell concatenation) to prevent injection
2. **Working directory**: Project root (found by walking up to `.git`)
3. **Timeout**: 30 seconds default; per-tool overrides allowed (e.g., 60s for `validate --all`)
4. **Signals**: SIGTERM on user abort, SIGKILL after grace period
5. **Output streams**: stdout captured and parsed as JSON; stderr captured for error diagnostics

### JSON Contract

Every Rootline response includes:
- `"version": 1` — required, validated on parse
- `"kind"` — tool identifier (e.g., `"rootline/query"`)
- Tool-specific data fields
- Optional `"truncated": true`, `"total"`, `"returned"` fields when output exceeds limits
- Optional `"_detail"` fields for auxiliary content not intended for model reasoning

Example:
```json
{
  "version": 1,
  "kind": "rootline/query",
  "truncated": true,
  "total": 500,
  "returned": 100,
  "data": {
    "meta": { "count": 100 },
    "rows": [ /* first 100 rows */ ]
  }
}
```

## Permissions Model

### Read-Only Tools (9 tools)

Execute immediately without confirmation. Safe for agent invocation.

- query, describe, validate, validate-all, tree, stats, graph, explain, analyze

**Runner behavior**: Execute and return result directly.

### Data-Mutating Tools (3 tools)

Modify Markdown document frontmatter or body. Require dry-run preview and user confirmation.

- fix (batch), set, repair apply

**Runner behavior**:
1. Auto-inject `--dry-run` if not present
2. Execute with `--dry-run`
3. Return preview with `confirmation_required: true`
4. Wait for user confirmation ("apply", "yes", etc.)
5. On approval, execute without `--dry-run`

### Schema-Mutating Tools (3 tools)

Modify `.stem` schema files. Require explicit typed confirmation.

- schema apply, new, migrate

**Runner behavior**:
1. Show `--dry-run` preview (if supported)
2. Request explicit approval (user must type "apply <signature>" or similar phrase)
3. On explicit approval, execute schema mutation
4. Return result

### Dry-Run Enforcement

The runner NEVER executes a mutating tool without first showing a preview. Schema mutations require typed (not abbreviated) confirmation.

## Compatibility Strategy

### Version Pinning

The extension declares a **minimum supported Rootline version** (e.g., v0.10.0+) and validates it at startup:

1. **Binary discovery**: Check `rootline --version`
2. **Version parsing**: Extract semantic version
3. **Validation**: Reject if version < minimum required
4. **Error handling**: Return `BINARY_NOT_FOUND` if version check fails

### Graceful Degradation

If the installed Rootline version does not support a required flag or command:

1. **Pre-flight validation**: Check tool availability before subprocess
2. **Unsupported tools**: Return error with code `INVALID_ARGUMENTS` and suggest upgrade
3. **Deprecated flags**: Detect flag removal and adjust command construction

### Backward Compatibility

Future Rootline CLI changes must not break Pi integration:

- JSON contracts are versioned (`"version": 1`)
- New fields are additive (never remove existing fields from JSON)
- Breaking command changes are avoided; if necessary, they require a major version bump and extension update

## Error Handling

All errors are classified by a named error code, enabling consistent handling:

| Code | Meaning | Recovery |
|------|---------|----------|
| `BINARY_NOT_FOUND` | rootline binary not found or version too old | Install/upgrade rootline |
| `PROJECT_ROOT_NOT_FOUND` | No `.git` directory; cannot establish context | Run from inside project |
| `TIMEOUT` | Command exceeded timeout (30s default) | Retry with filter/pagination or increase timeout |
| `ABORT` | User cancelled operation | Retry or discard |
| `PARSE_ERROR` | stdout is not valid JSON or missing `"version": 1` | Report Rootline bug |
| `ROOTLINE_ERROR` | Rootline exited with non-zero code | Inspect stderr; may be validation failure or missing file |
| `INVALID_ARGUMENTS` | Pre-flight validation detected invalid arguments | Check parameter names and formats |

**Error visibility**: Terminal errors (`BINARY_NOT_FOUND`, `PROJECT_ROOT_NOT_FOUND`, `PARSE_ERROR`) block further tool invocation. Transient errors (`TIMEOUT`) permit retry with adjusted parameters.

## Output Truncation

Large outputs are truncated at the JSON layer before rendering:

| Tool | Limit | Rationale |
|------|-------|-----------|
| query | 100 rows | Models reason over field counts; full datasets exceed context |
| validate-all | 50 errors, 50 files | Summarize remainder |
| tree | 6 depth levels, 50 children per node | Terminal readability |
| graph | 200 nodes, 500 edges | Interactive rendering capacity |
| stats | 20 distinct values per aggregation | Enum cardinality |
| describe | 100 fields | Schema comprehensibility |

When truncated, the response includes `"truncated": true`, `"total"`, and `"returned"` fields so the agent can decide to paginate, filter, or write full results to a temp file.

## Key Invariants

- **INV1**: The extension treats Rootline CLI JSON as the integration boundary; it does not import Go internal packages.
  - Verified: All code paths use subprocess + JSON parsing
- **INV2**: All mutating operations require user confirmation before execution.
  - Verified: Runner enforces dry-run preview + confirmation state machine
- **INV3**: Read-only tools are safe for agent invocation without confirmation.
  - Verified: Tool classification table and runner activation rules

## Alternatives Considered

### Direct Go Package Import
- **Rejected**: Rootline internals are not stable API; frequent refactoring would break extension

### Raw Shell String Invocation
- **Rejected**: Security risk; enables injection attacks; hard to validate arguments before execution

### HTTP Server + RPC
- **Rejected**: Adds deployment complexity; subprocess is simpler and doesn't require backgrounding

## Next Steps

Implementation tasks will follow this ADR:

1. **Package skeleton**: Set up directory structure and base types
2. **Shared runner**: Implement binary discovery, timeout, error handling
3. **Tool implementations**: Implement each tool class (read-only → data-mutating → schema-mutating)
4. **Tests**: Integration tests for each tool and permission class
5. **Documentation**: User guide, examples, troubleshooting
