# Schema Inspection and Document Scaffolding Reference

## describe

Use `describe` to show the effective `.stem` schema for exactly one path.

### Usage

```bash
rootline describe <path> -o json
rootline describe <path> -o table
rootline describe <dir> --field schema.id.next
rootline describe <path> --by-domain lifecycle_state -o json
```

`<path>` is required. It may be a file or directory.

### JSON Shape

```json
{
  "version": 1,
  "kind": "rootline/describe",
  "path": "docs/example/",
  "applies": ["docs/.stem", ".stem"],
  "schema": {
    "estado": {
      "type": "enum",
      "required": true,
      "values": ["Pending", "Completed"],
      "default": "Pending",
      "source": "docs/.stem"
    }
  },
  "validate": [],
  "derive": {},
  "aggregate": {},
  "links": {},
  "hints": []
}
```

For sequence fields, JSON may include `prefix`, `digits`, and `next`.

### Response Table

When explaining a schema, use this table:

```markdown
| Field | Type | Required | Values | Source |
|---|---|---:|---|---|
| estado | enum | yes | Pending, Completed | docs/.stem |
| id | sequence | yes | next: T005 | docs/.stem |
```

Then list, only when non-empty:

- validation rules: `<field>: <rule> (<severity>)`
- derived fields: `<field>: <expression>`
- aggregates: `<field>: <expression>`
- inherited schema chain from `applies`
- hints

## new

Use `new` to create one Markdown file from the effective schema of its parent directory.

### Usage

```bash
rootline new <file.md> --dry-run
rootline new <file.md>
rootline new <file.md> --force
```

`new` requires a file path. It does not choose directories or prompt for names.

### Deterministic Filename Rule

If the user gives a directory instead of a file:

1. Get the next sequence when available:
   ```bash
   rootline describe <dir> --field schema.id.next
   ```
2. Use a user-provided slug.
3. Build `<dir>/<ID>-<slug>.md` when `ID` exists; otherwise `<dir>/<slug>.md`.
4. If the slug is missing, ask for it before running `new`.

### Create Loop

```bash
rootline describe <dir> --field schema.id.next
rootline new <dir>/<ID>-<slug>.md --dry-run
rootline new <dir>/<ID>-<slug>.md
rootline validate <dir>/<ID>-<slug>.md -o json
git diff -- <dir>/<ID>-<slug>.md
```

The generated document:

- includes fields from the effective schema
- uses field defaults when defined
- leaves enum fields empty (with values as inline comment) when no explicit default exists
- writes required fields with empty values when needed
- derives the title from the filename

## Stem Resolution API (internal)

The central resolver in `internal/rules/resolver.go` exposes:
- `StemChain(path, root)` — stem files root→leaf
- `EffectiveSchema(path, root)` — merged schema with match filtering
- `Resolve(path, root)` — chain + schema + field provenance
- `(*Resolution).ClosestStem()` / `RootMostStem()` — explicit closest vs. root-most selection
- `ResolveLayered(path, root, monotonic bool)` — extends with `LayeredResolution.Layers` and `Conflicts`; in monotonic mode surfaces type widening, required loosening, enum extension, and structural loosening as conflicts

Use these instead of hand-rolling `WalkUp` + `entries[0]` indexing in new command code.

## Describe / Explain Provenance

`rootline describe` and `rootline explain` JSON output now includes:
- `layers` (array of strings) — ordered `.stem` chain root→leaf
- `provenance` (object) — field name → `.stem` path that last defined it

## Schema Commands

### schema propose

`rootline schema propose <dir> [--incremental] [--output json|table]` generates read-only schema proposals:
- Detects whether the directory has a hierarchical structure (E##/F##/S###/T### patterns) and calls `GenerateHierarchicalSchema` or `GenerateFlatSchema`
- Emits JSON: version 1, kind `"rootline/schema-proposals"`, with `proposals` array (id, operation, target, confidence, requires_agent, patch_preview) and summary
- `--incremental`: skips proposals covered by existing `.stem` files
- Never creates, modifies, or deletes any file

### schema apply

`rootline schema apply --report <proposals.json> [--dry-run]` applies schema proposals to `.stem` files:
- Input kind must be `"rootline/schema-proposals"`, version must be 1 (else structured error)
- Skips proposals with `requires_agent: true`
- `create_stem` operation: calls `ScaffoldSchema` to create the target `.stem` file
- `update_stem` operation: calls `ApplySchemaInferences` on the target `.stem` file
- `--dry-run`: no files written; reports what would be done
- Post-apply: runs `rootline validate --all` and includes results in output
- Emits JSON: version 1, kind `"rootline/schema-apply"` with applied/skipped/errors/validation_summary

## Fix All Schema Safety

`fix --all` now applies **data-only repairs** only (correct_value, add_field, migrate_value):
- Schema-surface proposals (extend_enum, add_aggregate, remove_stem_field) are skipped
- Skipped proposals appear in `schema_suggestions` count in JSON output and table note
- Use `rootline schema propose` or manually edit `.stem` files for schema changes

## Repair Apply Command

`rootline repair apply --report <analyze-report.json> [--dry-run]` applies data-only repairs:
- Accepts repair-surface proposals (correct_value, add_field, migrate_value, etc.)
- Silently rejects schema proposals (extend_enum, add_aggregate, remove_stem_field, etc.)
- Never creates, deletes, or modifies `.stem` files
- Supports `--dry-run` for preview without writes
- Emits JSON: version 1, kind "rootline/repair", with Changed/Skipped/Rejected/Errors

Engine: `internal/fix/repair.go` → `ApplyRepair(proposals, dryRun, root)`

## Schema Generation Services (internal)

`internal/infer/schema_gen.go` exports reusable services for schema candidate generation without file writes:
- `GenerateFlatSchema(ctx, dir, records, opts InferOptions) (*rules.StemFile, error)`
- `GenerateHierarchicalSchema(ctx, dir, records, opts InferOptions) (map[string]*rules.StemFile, error)`
- `DefaultInferOptions()` — `{SectionThreshold: 0.80, IncludeStructural: true}` (matches `init` defaults)

`init` command delegates to these instead of inline logic. Use in tests to generate schema candidates without filesystem writes.

## Proposal Surface Taxonomy

`internal/proposal/surface.go` classifies proposals without needing command context:

```go
Surface() ProposalSurface  // returns one of:
// schema       — mutates .stem (extend_enum, add_aggregate, remove_stem_field)
// repair       — mutates Markdown only (correct_value, add_field, ...)
// bootstrap    — scaffold missing .stem (missing_schema)
// migration    — bulk rename/type change
// diagnostic   — read-only governance finding
// requires_agent — needs human/agent decision
```

Use `proposal.Surface()` to gate schema vs. repair apply paths.

## Legacy Apply Deprecation

`rootline apply` is **deprecated**. On every invocation it prints to stderr:
> Warning: 'rootline apply' is deprecated. Use 'rootline schema apply' for schema changes or 'rootline repair apply' for data repairs.

The command remains functional for backward compatibility. `analyze` help text now references the replacement workflows. Replace scripts/agents using `apply` with:
- `rootline schema apply --report <proposals.json>` — schema changes (`.stem` files)
- `rootline repair apply --report <analyze-report.json>` — data corrections (frontmatter)

## Field Extraction — Array Projection

The global `--field` flag supports both object paths and array projection:

```bash
rootline query --from dir --field "rows[].path"          # extract path from each row
rootline query --from dir --field "rows[].frontmatter.estado"  # nested field per row
rootline graph dir --field "edges[].source"              # source of each graph edge
```

Syntax: append `[]` to a key to project over an array: `key[].subpath`. Returns a JSON array of extracted values. Errors on missing keys or non-array traversal.

Engine: `cmd/rootline/validate.go` → `extractField` → `extractFieldPath` (recursive).

## Apply Schema Proposals (internal)

`internal/fix/fix.go` exports `ApplySchemaProposals(ctx, report, root)`:
- Applies schema-surface proposals from either `report.Proposals` or `report.SchemaSuggestions`
- Currently handles: `extend_enum` (finds and rewrites the `.stem` file)
- Counterpart to `ApplyProposals` for the schema write path
- Used by e2e tests and callers that need to apply schema proposals separately from data proposals
