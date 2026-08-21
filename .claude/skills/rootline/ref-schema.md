# Schema Inspection and Document Scaffolding Reference

## describe

Use `describe` to show the effective `.stem` schema for exactly one path.

### Usage

```bash
rootline describe <path> -o json
rootline describe <path> -o table
rootline describe <dir> --field schema.id.next
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
    "notes": {
      "type": "string",
      "required": true,
      "source": "body.section[\"## Notes\"]",
      "defined_in": "docs/.stem"
    },
    "estado": {
      "type": "enum",
      "required": true,
      "values": ["Pending", "Completed"],
      "default": "Pending",
      "defined_in": "docs/.stem"
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
| Field | Type | Required | Values | Defined In | Source |
|---|---|---:|---|---|---|
| notes | string | yes |  | docs/.stem | body.section["## Notes"] |
| estado | enum | yes | Pending, Completed | docs/.stem |  |
| id | sequence | yes | next: T005 | docs/.stem |  |
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
- writes enum fields only when a default exists or the field is required
- Required enum fields without a default cause `rootline new` to refuse before dry-run output or disk write; the internal prospective renderer does not invent the first allowed value and validation reports the missing required value
- optional enum fields with no default are omitted
- does not invent the first allowed value
- writes required non-enum fields with empty values when needed
- derives the title from the filename

### Schemas multi-patrón: `next` vs `next_by_pattern`

Cuando `id` define múltiples patrones vía `match` (ej: `O*` y `T*`), el campo `next` retorna el próximo valor del primer patrón alfabético que coincide con entries existentes en el directorio — determinístico, pero incompleto para schemas multi-patrón.

Para obtener el próximo valor de **cada** patrón:

```bash
rootline describe <dir> --field schema.id.next_by_pattern
# → {"O*": "O14", "T*": "T014"}

rootline describe <dir>/O14-slug/ --field schema.id.next_by_pattern
# → {"T*": "T001"}
```

Usar `next_by_pattern` cuando el LLM necesita asignar números tanto para Outcomes como para Tasks en el mismo flujo de materialización.

## Stem Resolution API (internal)

The central resolver in `internal/rules/resolver.go` exposes:
- `StemChain(path, root)` — stem files root→leaf
- `EffectiveSchema(path, root)` — merged schema with match filtering
- `Resolve(path, root)` — chain + schema + field provenance
- `(*Resolution).ClosestStem()` / `RootMostStem()` — explicit closest vs. root-most selection
- `ResolveLayered(path, root)` — returns cumulative `LayeredResolution.Layers` and `Conflicts`; monotonic compatibility is always enforced

Use these instead of hand-rolling `WalkUp` + `entries[0]` indexing in new command code.

## Describe / Explain Provenance

`rootline describe` and `rootline explain` JSON output includes:
- `layers` (array of strings) — ordered `.stem` chain root→leaf
- `provenance` (object) — field name → `.stem` path that last defined it
- `source` — a logical body extraction directive when the field has one
- `defined_in` — the physical `.stem` that declares the field

A source-backed field uses a real type plus `source: body.section["## Heading"]`; frontmatter is an override. Child omission inherits the stable source binding. `new` and `migrate --scaffold` materialize missing required sections in lexical heading order using a non-empty default or `<!-- TODO -->`.

Author required section-backed fields in `.stem` like this; `defined_in` appears only in command output, not in authored declarations:

```yaml
notes:
  type: string
  required: true
  source: body.section["## Notes"]
```

## Schema Commands

### schema propose

`rootline schema propose <dir> [--incremental] [--output json|table]` generates read-only schema proposals:
- Detects whether the directory has a hierarchical structure (E##/F##/S###/T### patterns) and calls `GenerateHierarchicalSchema` or `GenerateFlatSchema`
- Emits JSON: version 1, kind `"rootline/schema-proposals"`, with `proposals` array (id, operation, target, confidence, requires_agent, `patch`, `patch_preview`) and summary
- `patch` carries the **full** inferred YAML and is what `schema apply` writes. `patch_preview` is the same YAML truncated to 200 chars, for display only — never apply from it
- `root` carries the **absolute** scan root, so the report stays applicable from any working directory
- `--incremental`: skips proposals covered by existing `.stem` files
- Never creates, modifies, or deletes any file

### schema apply

`rootline schema apply --report <proposals.json> [--dry-run] [--force]` applies schema proposals or analyze-derived schema inferences to `.stem` files:
- Input kind must be `"rootline/schema-proposals"`, `"rootline/analyze"`, or legacy `"analyze"`; report version must be 1 (else structured error)
- Skips proposals with `requires_agent: true`
- `create_stem` operation: writes `proposal.patch` **byte-identical** to the target `.stem`. Apply never re-derives the schema, so what a reviewer approved is what lands on disk
- A proposal with an empty or missing `patch` is refused into `errors[]` with an instruction to re-run `schema propose`; it is never silently scaffolded into untyped `type: string` shells
- Analyze reports are planned in memory first, then pass through the same prospective hierarchy gate as proposal reports before any dry-run action is published or any file is written
- Unknown operations: rejected with message in `rejected[]` (policy refusal, not error)
- **Flag: `--force`** — Overwrites existing `.stem` files when applying `create_stem` proposals. Without `--force`, proposals targeting existing files are rejected (policy refusal).
- `--dry-run`: no files written; reports the same accepted actions and `stem_health[]` governance verdicts the real write path would use
- Scan root: `report.root` when present, else `report.path` resolved against the caller's CWD (legacy reports)
- Before publishing `applied[]` actions or writing files, apply validates the complete virtual `.stem` hierarchy produced by the whole batch. Error-severity `stem_health[]` diagnostics block with `complete:false`, non-zero exit, and `applied:[]`; warning/info diagnostics remain visible and nonblocking
- Writes use atomic per-file replacement. In a multi-file batch, each accepted file is replaced atomically on its own; if a write fails after earlier files succeeded, apply records the error and continues best-effort for remaining files rather than rolling back prior successful files
- Accepted writes are validated and replaced through one bound physical target. Internal aliases are supported, including symlinked parents inside the scan root, but aliases that physically escape the scan root are rejected before any write is attempted
- Post-apply: runs `rootline validate --all` against that scan root. A failed scan surfaces in `errors[]` — it is never reported as an all-zero `validation_summary`, which would be indistinguishable from a clean run. Document invalidity stays separate from apply governance: a governance-valid apply can return `complete:true` while `validation_summary.invalid_files` is non-zero
- Emits JSON: version 1, kind `"rootline/schema-apply"` with applied/skipped/rejected/errors/validation_summary and non-omitempty `stem_health[]` (serialized as `[]` when empty)
- Table output renders a `Stem Health` section only when diagnostics are non-empty, with columns `Path`, `Check`, `Field`, `Severity`, and `Message`

**Path containment.** Each `create_stem` target is validated with
`fix.ContainPath(scanRoot, target, fix.PolicyAcceptAbsolute)` before anything is written, and the
write targets the *validated* path, not the raw one.

Two deliberate differences from `repair apply`:
- **Absolute targets are accepted, then confined** (`PolicyAcceptAbsolute`). `schema propose`
  emits absolute `.stem` targets, so the propose→apply contract needs them to keep working.
  `repair apply` uses `PolicyRejectAbsolute` because its report paths are root-relative by
  construction.
- **Policy refusals go to `rejected[]`** (overwrite without `--force`, unknown operations).
  Containment violations go to `errors[]` (operational failures).

`--dry-run` adds the same additive `resolved_targets: {accepted, rejected}` envelope that
`repair apply` emits (`fix.ResolvedTargetsBreakdown`); omitted otherwise, contract stays
version 1. `fix.ContainmentReason(err)` is exported so both apply commands unwrap
`ContainmentError` through one implementation.

## Fix All Schema Safety

`fix --all` now applies **data-only repairs** only (correct_value, add_field, migrate_value):
- Schema-surface proposals (extend_enum, add_aggregate, remove_stem_field) are skipped
- Skipped proposals appear in `schema_suggestions` count in JSON output and table note
- Use `rootline schema propose` or manually edit `.stem` files for schema changes

## Repair Apply Command

`rootline repair apply --report <repairs.json> [--dry-run]` applies data-only repairs (from `rootline fix --all <dir> --dry-run -o json`):
- Accepts repair-surface proposals (correct_value, add_field, migrate_value, etc.)
- Rejects schema proposals (extend_enum, add_aggregate, remove_stem_field, etc.); appear in output under Rejected
- Never creates, deletes, or modifies `.stem` files
- Supports `--dry-run` for preview without writes
- Emits JSON: version 1, kind "rootline/repair", with Changed/Skipped/Rejected/Errors

**Path containment.** Report paths are untrusted. Every path is checked against the scan root
before any file is opened, so an escaping (`../..`) or absolute path is never read or written.
Violations land in `rejected[]` (a policy refusal), not `errors[]` (a failed write). A proposal
is applied whole or not at all — one escaping path discards the whole proposal.

With `--dry-run` the result adds `resolved_targets: {accepted: {reportPath: absPath}, rejected:
{reportPath: reason}}`, so a caller can see where writes would land and why any were refused.
Additive and omitted outside dry-run; the contract stays version 1.

Engine: `internal/fix/repair.go` → `ApplyRepair(proposals, dryRun, root)`;
guard: `internal/fix/contain.go` → `ContainPath(root, path, policy)`.
`applySetSection` is reachable from both `repair apply` and `fix --all`, so it takes a
`ContainmentPolicy` and checks containment itself: `PolicyRejectAbsolute` for report-supplied
paths, `PolicyAcceptAbsolute` for scan-derived `fix --all` paths.

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

## Legacy Apply Removal

`rootline apply` no longer exists — invoking it fails with `unknown command "apply"`. Replace scripts/agents that used `apply` with:
- `rootline schema apply --report <proposals.json>` — schema changes (`.stem` files)
- `rootline repair apply --report <repairs.json>` — data corrections (frontmatter)

## Field Extraction — Array Projection

The global `--field` flag supports both object paths and array projection:

```bash
rootline query --from dir --field "rows[].path"          # extract path from each row
rootline query --from dir --field "rows[].frontmatter.estado"  # nested field per row
rootline graph dir --field "edges[].source"              # source of each graph edge
```

Syntax: append `[]` to a key to project over an array: `key[].subpath`. Returns a JSON array of extracted values. Errors on missing keys or non-array traversal.

`--field` requires `--output json` and is repeatable — one path returns the bare value, several return a JSON array in flag order:

```bash
rootline query dir --count --field kind --field meta.count   # ["rootline/count",6]
rootline stats dir -o table --field kind                     # rc=1: --field requires --output json
```

Engine: `cmd/rootline/validate.go` → `extractFields` → `extractFieldPath` (recursive).

## Apply Schema Proposals (internal)

`internal/fix/fix.go` exports `ApplySchemaProposals(ctx, report, root)`:
- Applies schema-surface proposals from either `report.Proposals` or `report.SchemaSuggestions`
- Currently handles: `extend_enum` (finds and rewrites the `.stem` file)
- Counterpart to `ApplyProposals` for the schema write path
- Used by e2e tests and callers that need to apply schema proposals separately from data proposals
