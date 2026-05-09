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
- uses the first enum value when no default exists
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
