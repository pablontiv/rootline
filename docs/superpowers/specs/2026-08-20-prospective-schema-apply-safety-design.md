---
estado: Specified
---

# Prospective Schema Apply Safety Design

**Date:** 2026-08-20
**Status:** Approved design
**Parent change:** PR #193
**Follow-up enhancement:** #194

## Purpose

Make both `schema apply` report paths validate the complete prospective governance hierarchy before they publish actions or write `.stem` files, while preserving Rootline's existing per-file best-effort write contract.

PR #193 validates proposed field declarations before writing, but the proposal path currently validates each candidate `.stem` in isolation. A locally valid child can therefore widen an inherited field, be reported as `complete: true`, and only fail stem health on the next `validate --all`. The proposal path and analyze/inference path also write `.stem` files without the atomic replacement guarantee used elsewhere in Rootline.

## Decisions

1. Both `rootline/schema-proposals` and `rootline/analyze` report paths use one prospective governance validation pipeline.
2. The pipeline evaluates the complete virtual `.stem` hierarchy after applying every accepted candidate in memory.
3. Stem-health diagnostics with severity `error` block the entire plan before any write or dry-run publication. `warn` and `info` diagnostics do not block.
4. Dry-run and real execution produce the same planning and validation verdict; they differ only in filesystem writes and post-write observations.
5. `.stem` replacement is atomic per file through `fsx.WriteFileAtomic`.
6. Multi-file execution remains best-effort, not all-or-nothing. Earlier successful replacements remain in place when a later write fails.
7. Write failure does not stop later independent writes unless the context is canceled.
8. The existing post-apply document validation remains separate from governance validation. Document invalidity does not retroactively mean the governance declaration is malformed.
9. Diagnostics and actions are deterministic regardless of map or report insertion order.
10. A general injectable filesystem abstraction is deferred to enhancement #194 and is not required by this fix.

## Scope

### In scope

- Pure representation and evaluation of discovered `.stem` state.
- Virtual overlay of all accepted candidates in one batch.
- Full declaration, inheritance, monotonicity, and stem-health evaluation before execution.
- Shared planning across schema-proposals and analyze/inference reports.
- Atomic `.stem` replacement in both paths.
- Machine-readable prospective stem-health diagnostics.
- Deterministic errors, actions, and target ordering.
- Regression tests for cross-layer invalidity, dry-run parity, and interrupted writes.
- Documentation of the per-file best-effort contract.

### Out of scope

- A multi-file filesystem transaction, journal, or rollback protocol.
- Blocking on stem-health warnings or informational diagnostics.
- Changing document-validation semantics after schema apply.
- Changing schema merge, match, monotonicity, or provenance rules.
- A writable virtual filesystem.
- General filesystem injection across `WalkUp`, `Resolve`, and discovery; tracked by #194.
- New schema report kinds or envelope versions.

## Architecture

### Prospective stem state

`internal/rules` owns an immutable logical snapshot named `StemState` with three required components:

```go
type StemState struct {
    Root        string
    Stems       map[string]*StemFile
    ParseErrors map[string]error
    Entries     map[string]StemStateEntry
}

type StemStateEntry struct {
    IsDir bool
}
```

`Root` is the normalized absolute governance root. `Stems` is keyed by normalized absolute `.stem` path. `ParseErrors` preserves parse failures by normalized absolute path so evaluation can emit the existing `yaml-valid` diagnostics instead of aborting discovery. `Entries` contains every discovered file and directory path needed by checks such as `scope-match`. Diagnostic paths are always derived relative to `Root`.

Discovery records malformed real stems in `ParseErrors` and continues. A proposed overlay must parse successfully before entering the virtual state; candidate parse failures remain blocking planning errors.

Discovery from the real filesystem produces an initial state. Overlay operations return a new state or a private mutable planning copy; callers cannot mutate the discovered baseline accidentally.

### Pure stem-health evaluator

Stem-health processing is split into two layers:

1. `DiscoverStemState(ctx, root)` reads the real filesystem and parses stems.
2. `EvaluateStemState(ctx, state)` evaluates the supplied state without filesystem reads or writes.

`ValidateStemHealth` remains the compatible production facade:

```go
func ValidateStemHealth(ctx context.Context, root string) (*StemHealthResult, error) {
    state, err := DiscoverStemState(ctx, root)
    if err != nil {
        return nil, err
    }
    return EvaluateStemState(ctx, state)
}
```

The evaluator reuses the existing declaration and compatibility authorities. It must not duplicate monotonic rules in a schema-apply package. Existing checks that currently call `WalkUp` or read directories are refactored to consume the state inventory and state-derived chains.

### Shared write plan

Both report paths produce the same internal operation type:

```go
type StemWritePlan struct {
    Target       string
    Content      []byte
    Action       string
    ReportTarget string
}
```

The plan contains complete serialized bytes but has no write capability. Containment, policy rejection, skipped proposals, and target existence are resolved before a candidate enters the plan.

All plan candidates are overlaid onto one `StemState`. Evaluating proposals separately is forbidden because two individually valid operations may be incompatible together.

### Inference planning

`internal/infer.ApplySchemaInferences` currently combines YAML transformation and persistence. It is split into:

```go
func PlanSchemaInferences(stemPath string, inferences []ReportInference) (*SchemaInferencePlan, error)
func ApplySchemaInferencePlan(plan *SchemaInferencePlan) error
```

`SchemaInferencePlan` contains serialized candidate bytes plus applied, skipped, and rejected actions. `ApplySchemaInferencePlan` persists through `fsx.WriteFileAtomic`. `ApplySchemaInferences` remains as a compatibility wrapper that delegates to these two functions and omits persistence in dry-run mode.

`schema apply` calls the planning function directly so analyze-derived changes participate in the same batch-wide virtual validation as proposal-derived changes.

### Shared executor

A common executor accepts only a prospectively validated plan. It sorts operations deterministically and calls:

```go
fsx.WriteFileAtomic(target, content, 0o644)
```

The helper preserves the mode of existing files and uses the requested mode only for new files. Each failed target remains byte-identical and leaves no staging debris. Execution continues to later targets unless `ctx.Err()` is non-nil.

The executor is not a transaction coordinator. It reports exactly which operations succeeded and which failed.

## Data Flow

```text
report bytes
  -> report kind/version validation
  -> root resolution and containment
  -> path-specific candidate planning
  -> complete []StemWritePlan
  -> DiscoverStemState(real root)
  -> overlay every candidate
  -> EvaluateStemState(virtual state)
  -> classify diagnostics
       error present -> reject before writes
       warn/info only -> validated plan
  -> dry-run: emit plan and diagnostics
  -> real: atomic best-effort execution
  -> post-apply document validation
  -> seal and emit envelope
```

The real run must not repeat planning with a different code path. It executes the exact plan whose virtual state passed evaluation.

## Error and Envelope Contract

`SchemaApplyResult` gains an always-initialized `stem_health` collection containing non-passing prospective diagnostics. This is an additive field in the existing version 1 envelope; no version bump is introduced. Contract tests must assert that the key is present as `[]` when no diagnostics exist.

### Blocking prospective failure

A declaration error, monotonic conflict, invalid inherited state, or any other stem-health `error` produces:

- non-zero exit;
- `complete: false`;
- `applied: []`;
- no filesystem writes;
- deterministic `errors[]` preserving path, check, field, and message;
- the same verdict under `--dry-run`.

Warnings and informational diagnostics remain in `stem_health[]` but do not fail the command.

### Write failure

A per-file write failure produces:

- the target's original bytes and mode unchanged;
- no staging debris;
- `complete: false`;
- an entry in `errors[]` naming the report target;
- prior successful operations retained;
- later independent operations attempted unless canceled.

### Post-apply document invalidity

The existing `validation_summary` continues to report whether records satisfy the resulting schema. Invalid records do not imply a malformed schema and do not erase successful writes. Governance-invalid states are prevented earlier by prospective stem-state evaluation.

## Determinism

Before evaluation or emission:

- targets are sorted by normalized path;
- stem files are evaluated in root-to-leaf path order;
- diagnostics are sorted by path, check, field, and message;
- action lists use target order;
- map insertion order and report category order cannot change the envelope.

Duplicate plans for one target are resolved during planning using the existing overwrite policy. The virtual state reflects the exact final content that execution would leave at that target.

## Testing Strategy

### `internal/rules`

- Real discovery plus evaluation matches the current `ValidateStemHealth` diagnostics.
- Parent `enum` to child `string` produces a blocking error.
- Parent `string` to child `enum` remains valid narrowing.
- Required, severity, enum-domain, source-binding, and structural loosening retain existing verdicts.
- Warnings and informational findings remain non-blocking.
- Multiple overlays are evaluated together.
- Different map insertion orders produce byte-equivalent diagnostics.
- Context cancellation propagates.

### Schema-proposals path

- The PR #193 reproduction is permanent: parent enum plus proposed child string is rejected before write.
- Dry-run and real execution return the same blocking diagnostics and neither creates the child `.stem`.
- Two candidates that are locally valid but jointly invalid reject the entire plan.
- Containment and overwrite refusals keep their current classification.
- An atomic write failure preserves the original target and cleans staging files.
- A multi-target run retains earlier successes and attempts later writes after one failure.

### Analyze/inference path

- Inference planning returns bytes without writing.
- Analyze candidates participate in the same hierarchy validation.
- YAML comments, native scalar types, field source identity, and existing file modes remain preserved.
- Dry-run and real execution report the same planned actions.
- Blocking governance errors publish no actions and perform no writes.

### Contracts and integration

- JSON and table outputs cover empty and populated `stem_health` collections.
- Documentation describes blocking severities and best-effort atomic writes.
- `just check`, `just test`, `just coverage-check`, and `go vet ./...` pass.
- Roadmap and design/spec validation pass.
- Every package remains at or above its configured 85% coverage floor.

## Acceptance Criteria

1. Neither schema-apply report path reports or writes a prospective state containing a stem-health error.
2. The parent-enum/child-string reproduction fails before publication in dry-run and real modes.
3. All accepted candidates in a report are evaluated as one virtual hierarchy.
4. Dry-run and execution differ only in I/O and post-write observations.
5. Every `.stem` write in both paths uses atomic per-file replacement and preserves existing modes.
6. Multi-file behavior remains explicitly best-effort and accurately reported.
7. `ValidateStemHealth` and prospective apply consume one stem-health authority.
8. Diagnostics and actions are deterministic.
9. The machine-readable envelope documents prospective diagnostics.
10. Coverage and repository validation gates remain green.

## Deferred Enhancement

Issue #194 tracks a read-only injectable filesystem for discovery, walk-up resolution, parsing, layered resolution, and provenance. It has an adoption gate: implementation requires at least one real consumer in addition to `schema apply`. The abstraction must be introduced incrementally behind existing APIs and must not replace the pure `StemState` evaluator defined here.
