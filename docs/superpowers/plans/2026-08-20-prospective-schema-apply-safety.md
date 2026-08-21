---
estado: Specified
---

# Prospective Schema Apply Safety Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make both `schema apply` report paths reject batch-wide governance errors before publication and replace every accepted `.stem` atomically per file.

**Architecture:** `internal/rules` gains a discoverable, overlayable `StemState` and a pure stem-health evaluator. Both schema report paths produce the same internal write-plan shape, overlay the complete batch on one state, reject any error-severity diagnostic, and only then execute atomic per-file writes while preserving Rootline's best-effort multi-file contract.

**Tech Stack:** Go 1.26+, standard library, `gopkg.in/yaml.v3`, Cobra, existing `internal/fsx`, standard `testing` package.

**Spec:** `docs/superpowers/specs/2026-08-20-prospective-schema-apply-safety-design.md`

## Global Constraints

- Both `rootline/schema-proposals` and `rootline/analyze` inputs use one prospective governance evaluator.
- Evaluate every accepted candidate as one virtual hierarchy; never validate candidates independently as the final safety verdict.
- Stem-health severity `error` blocks the complete plan before writes and before dry-run actions are published; `warn` and `info` do not block.
- Dry-run and real execution share planning and prospective validation; they differ only in writes and post-write document validation.
- Use `fsx.WriteFileAtomic` for every `.stem` replacement in both report paths.
- Preserve atomicity per file and the existing best-effort multi-file contract; do not add a global rollback or journal.
- Continue after an independent write failure unless `ctx.Err()` is non-nil.
- Preserve existing file modes; use `0o644` only as the create-new default.
- `SchemaApplyResult` remains version 1 and always emits `stem_health: []` when there are no prospective diagnostics.
- Sort targets, actions, and diagnostics deterministically.
- Preserve the public `ApplySchemaInferences` API as a compatibility wrapper.
- Do not add dependencies or implement the injectable filesystem tracked by #194.
- Keep every package at or above its configured 85% coverage floor.

## File Map

### New files

- `internal/rules/stem_state.go` — immutable discovered/overlaid stem state, entry inventory, and state-local chain construction.
- `internal/rules/stem_state_test.go` — discovery, parse-error retention, overlay immutability, root-boundary, and ordering tests.
- `internal/rules/stem_state_health_test.go` — pure evaluator parity, cross-layer conflicts, warning policy, and determinism tests.
- `internal/infer/apply_plan.go` — inference plan DTO plus atomic persistence wrapper.
- `internal/infer/apply_plan_test.go` — no-write planning, byte preservation, mode preservation, and wrapper parity tests.
- `cmd/rootline/schema_apply_execution.go` — shared batch-plan validation and atomic best-effort execution.
- `cmd/rootline/schema_apply_execution_test.go` — blocking diagnostics, non-blocking warnings, deterministic actions, and injected write-failure tests.
- `cmd/rootline/schema_apply_hierarchy_test.go` — CLI regressions for proposal and analyze hierarchy safety.

### Modified files

- `internal/rules/stemhealth.go` — retain public types/facade; move evaluation from direct filesystem calls to `StemState`.
- `internal/rules/resolver.go` — factor resolution from an explicit `[]StemEntry` chain so real and virtual state share monotonic logic.
- `internal/rules/stemhealth_test.go` — preserve current facade behavior and adjust helpers to the state-backed evaluator.
- `internal/rules/resolver_test.go` — prove explicit-chain and filesystem resolution parity.
- `internal/infer/apply.go` — keep YAML node mutation logic; delegate planning/persistence to `apply_plan.go`.
- `internal/infer/apply_test.go` — preserve compatibility-wrapper behavior.
- `internal/e2e/schema_apply_e2e_test.go` — retain public inference API coverage after the split.
- `cmd/rootline/schema.go` — use shared plans for both report kinds, emit `stem_health`, and remove direct `os.WriteFile` calls.
- `cmd/rootline/schema_apply_planner_test.go` — adapt proposal planner assertions to the shared plan type.
- `cmd/rootline/schema_apply_preflight_test.go` — preserve current declaration/preflight safety cases.
- `cmd/rootline/schema_test.go` — assert version-1 envelope compatibility and post-apply document semantics.
- `.claude/skills/rootline/ref-schema.md` — document prospective hierarchy validation, diagnostics, and atomic best-effort writes.
- `docs/analyze.md` — document that analyze-derived schema changes use the same prospective safety gate.
- `CHANGELOG.md` — record the publication-safety correction.

---

### Task 1: Model and discover prospective stem state

**Files:**
- Create: `internal/rules/stem_state.go`
- Create: `internal/rules/stem_state_test.go`

**Interfaces:**
- Consumes: `ParseStem(path, content)`, `WalkUp(path)`, `StemEntry`, `StemFile`.
- Produces:

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

func DiscoverStemState(ctx context.Context, root string) (*StemState, error)
func (s *StemState) Overlay(path string, content []byte) (*StemState, error)
func (s *StemState) Chain(path string) []StemEntry
func (s *StemState) EvaluatedStemPaths() []string
func (s *StemState) MatchingFiles(dir, pattern string) []string
```

- `Overlay` returns a deep-enough clone of all maps; it never mutates its receiver.
- `Chain` returns root-to-leaf entries and stops after the closest `root: true` marker.
- `EvaluatedStemPaths` includes only `.stem` paths at or below `Root`; ancestor stems are resolution context, not diagnostics owned by this scan.

- [ ] **Step 1: Write discovery and parse-error tests**

Add tests that seed a valid root stem, malformed child stem, nested records, and ignored `.git`/`node_modules` content:

```go
func TestDiscoverStemStatePreservesParseErrorsAndInventory(t *testing.T) {
    root := t.TempDir()
    mustWriteStemStateFile(t, filepath.Join(root, ".stem"), "version: 2\nroot: true\n")
    mustWriteStemStateFile(t, filepath.Join(root, "docs", ".stem"), "version: [broken\n")
    mustWriteStemStateFile(t, filepath.Join(root, "docs", "a.md"), "# A\n")
    mustWriteStemStateFile(t, filepath.Join(root, ".git", ".stem"), "version: 2\n")

    state, err := DiscoverStemState(context.Background(), root)
    if err != nil {
        t.Fatal(err)
    }
    if state.Stems[filepath.Join(root, ".stem")] == nil {
        t.Fatal("root stem was not parsed")
    }
    if state.ParseErrors[filepath.Join(root, "docs", ".stem")] == nil {
        t.Fatal("malformed stem parse error was not retained")
    }
    if _, ok := state.Entries[filepath.Join(root, "docs", "a.md")]; !ok {
        t.Fatal("record inventory is incomplete")
    }
    if _, ok := state.Entries[filepath.Join(root, ".git", ".stem")]; ok {
        t.Fatal(".git must be excluded")
    }
}
```

Also add `TestDiscoverStemStateIncludesExternalAncestorContext` to scan a child directory governed by a parent `.stem` outside `Root` and assert that the parent is present in `Stems` but absent from `EvaluatedStemPaths()`.

- [ ] **Step 2: Run the discovery tests and verify RED**

Run:

```bash
go test ./internal/rules -run 'TestDiscoverStemState' -count=1
```

Expected: compilation failure because `DiscoverStemState`, `StemState`, and `StemStateEntry` do not exist.

- [ ] **Step 3: Implement state discovery**

Implement `DiscoverStemState` with `filepath.Abs`, `filepath.Clean`, one `filepath.Walk` under `Root`, and `WalkUp(root)` for external ancestor context. Preserve malformed descendants in `ParseErrors` instead of returning their parse error. Return traversal/context errors and `ctx.Err()`.

Use a single helper for parsing:

```go
func addStemToState(state *StemState, path string, content []byte) {
    stem, err := ParseStem(path, content)
    if err != nil {
        state.ParseErrors[path] = err
        delete(state.Stems, path)
        return
    }
    state.Stems[path] = stem
    delete(state.ParseErrors, path)
}
```

- [ ] **Step 4: Write overlay, chain, and deterministic-order tests**

Add:

```go
func TestStemStateOverlayIsImmutableAndClearsParseError(t *testing.T)
func TestStemStateChainStopsAtNestedRootMarker(t *testing.T)
func TestStemStateEvaluatedPathsAreSorted(t *testing.T)
func TestStemStateMatchingFilesUsesImmediateNonDirectoryEntries(t *testing.T)
```

The overlay test must start with a malformed `.stem`, overlay valid bytes, assert the clone contains the parsed stem, and assert the original still contains its parse error.

- [ ] **Step 5: Run the new tests and verify RED**

Run:

```bash
go test ./internal/rules -run 'TestStemState' -count=1
```

Expected: failures for missing `Overlay`, `Chain`, ordering, and matching behavior.

- [ ] **Step 6: Implement overlay and state-local lookup**

Clone maps explicitly, parse candidate bytes through `ParseStem`, add the target and parent directory to `Entries`, and return a contextual error without mutating either state when parsing fails.

Build `Chain` by walking directories upward through the state maps, collecting `.stem` entries, stopping at `Root` or the first parsed stem with `Root == true`, then reversing the collected slice.

- [ ] **Step 7: Run focused and package tests**

Run:

```bash
go test ./internal/rules -run 'Test(DiscoverStemState|StemState)' -count=1
go test ./internal/rules -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit Task 1**

```bash
git add internal/rules/stem_state.go internal/rules/stem_state_test.go
git commit -m "refactor(rules): model prospective stem state"
```

- [ ] **Step 9: Request task review**

Request a read-only review of the Task 1 commit against the `StemState` interfaces and fix all Critical/Important findings before Task 2.

---

### Task 2: Evaluate stem health from explicit state

**Files:**
- Create: `internal/rules/stem_state_health_test.go`
- Modify: `internal/rules/stemhealth.go`
- Modify: `internal/rules/resolver.go`
- Modify: `internal/rules/resolver_test.go`
- Modify: `internal/rules/stemhealth_test.go`

**Interfaces:**
- Consumes: Task 1 `StemState`, `StemState.Chain`, `StemState.MatchingFiles`.
- Produces:

```go
func EvaluateStemState(ctx context.Context, state *StemState) (*StemHealthResult, error)
func resolveFromEntries(path string, entries []StemEntry) (*Resolution, error)
func resolveLayeredFromEntries(path string, entries []StemEntry) (*LayeredResolution, error)
```

- `Resolve` and `ResolveLayered` remain public filesystem-backed facades.
- `ValidateStemHealth` remains public and becomes discovery plus evaluation.

- [ ] **Step 1: Write evaluator parity and hierarchy tests**

Add a fixture helper that calls both paths:

```go
func evaluateBoth(t *testing.T, root string) (*StemHealthResult, *StemHealthResult) {
    t.Helper()
    direct, err := ValidateStemHealth(context.Background(), root)
    if err != nil {
        t.Fatal(err)
    }
    state, err := DiscoverStemState(context.Background(), root)
    if err != nil {
        t.Fatal(err)
    }
    pure, err := EvaluateStemState(context.Background(), state)
    if err != nil {
        t.Fatal(err)
    }
    return direct, pure
}
```

Cover valid stems, malformed YAML, orphan scope, rule-field references, aggregates, unknown link keys, nested roots, and monotonic conflicts. Compare complete `StemHealthDiagnostics` slices, not only counts.

Add the exact hierarchy case:

```go
func TestEvaluateStemStateRejectsEnumToStringWidening(t *testing.T) {
    // root estado: enum [Pending, Done]
    // child estado: string
    // expect one error-severity type-consistency diagnostic owned by child/.stem
}
```

Add a warning-only state and assert it produces no error-severity diagnostic.

- [ ] **Step 2: Run evaluator tests and verify RED**

Run:

```bash
go test ./internal/rules -run 'TestEvaluateStemState' -count=1
```

Expected: compilation failure because `EvaluateStemState` does not exist.

- [ ] **Step 3: Factor resolver logic from explicit entries**

Move the existing merge, match filtering, provenance, layer collection, and cumulative monotonic checks behind `resolveFromEntries` and `resolveLayeredFromEntries`:

```go
func Resolve(path, root string) (*Resolution, error) {
    entries, err := WalkUp(path)
    if err != nil {
        return nil, err
    }
    return resolveFromEntries(path, entries)
}

func ResolveLayered(path, root string) (*LayeredResolution, error) {
    entries, err := WalkUp(path)
    if err != nil {
        return nil, err
    }
    return resolveLayeredFromEntries(path, entries)
}
```

Add tests proving filesystem and explicit-entry outputs have identical effective schemas, provenance, layers, and conflicts.

- [ ] **Step 4: Run resolver tests**

Run:

```bash
go test ./internal/rules -run 'TestResolve.*(Entries|Layered)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Move checks into `EvaluateStemState`**

Retain `StemHealthCheck`, `StemHealthResult`, `StemHealthDiagnostic`, and `StemHealthDiagnostics` in `stemhealth.go`. Replace direct calls as follows:

- `filepath.Walk` and `ParseStemFile` → `state.EvaluatedStemPaths`, `state.Stems`, `state.ParseErrors`.
- `os.ReadDir` → `state.MatchingFiles`.
- `WalkUp(dir)` → `state.Chain(dir)`.
- `ResolveLayered(dir, root)` → `resolveLayeredFromEntries(dir, state.Chain(dir))`.
- ancestor root lookup → state path traversal.

Preserve all existing check names, status values, messages, ownership rules, and context-cancellation points. Sort the final `checks` slice by path, check name, field, status, then message before returning.

Make the facade exact:

```go
func ValidateStemHealth(ctx context.Context, absRoot string) (*StemHealthResult, error) {
    state, err := DiscoverStemState(ctx, absRoot)
    if err != nil {
        return nil, err
    }
    return EvaluateStemState(ctx, state)
}
```

- [ ] **Step 6: Run evaluator parity tests and verify GREEN**

Run:

```bash
go test ./internal/rules -run 'Test(EvaluateStemState|ValidateStemHealth|Resolve)' -count=1
```

Expected: PASS with byte-equivalent diagnostics.

- [ ] **Step 7: Add determinism and cancellation tests**

Construct equivalent `StemState` values with opposite map insertion order and assert identical diagnostics. Cancel a context before evaluation and assert `context.Canceled`.

- [ ] **Step 8: Run full rules tests with race detection**

```bash
go test ./internal/rules -race -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit Task 2**

```bash
git add internal/rules/stemhealth.go internal/rules/stem_state_health_test.go internal/rules/stemhealth_test.go internal/rules/resolver.go internal/rules/resolver_test.go
git commit -m "refactor(rules): evaluate stem health from state"
```

- [ ] **Step 10: Request task review**

Request a read-only review focused on diagnostic parity, ancestor context, root-marker stopping, and duplicate monotonic diagnostics. Fix all Critical/Important findings.

---

### Task 3: Separate inference planning from persistence

**Files:**
- Create: `internal/infer/apply_plan.go`
- Create: `internal/infer/apply_plan_test.go`
- Modify: `internal/infer/apply.go`
- Modify: `internal/infer/apply_test.go`
- Modify: `internal/e2e/schema_apply_e2e_test.go`

**Interfaces:**
- Consumes: existing YAML node mutation helpers and `fsx.WriteFileAtomic`.
- Produces:

```go
type SchemaInferencePlan struct {
    Target   string
    Content  []byte
    Result   ApplyResult
    Modified bool
}

func PlanSchemaInferences(stemPath string, inferences []ReportInference) (*SchemaInferencePlan, error)
func ApplySchemaInferencePlan(plan *SchemaInferencePlan) error
```

- Existing `ApplySchemaInferences(stemPath, inferences, dryRun)` remains source-compatible.

- [ ] **Step 1: Write no-write planning tests**

Add:

```go
func TestPlanSchemaInferencesReturnsCandidateWithoutWriting(t *testing.T)
func TestPlanSchemaInferencesNoModificationReturnsOriginalTargetAndModifiedFalse(t *testing.T)
func TestPlanSchemaInferencesPreservesCommentsAndNativeScalars(t *testing.T)
```

The first test snapshots bytes and mode, calls `PlanSchemaInferences`, asserts the target is unchanged, and parses `plan.Content` to prove the intended inference is present.

- [ ] **Step 2: Run planner tests and verify RED**

```bash
go test ./internal/infer -run 'TestPlanSchemaInferences' -count=1
```

Expected: compilation failure because the plan type and function do not exist.

- [ ] **Step 3: Extract the existing transform phase**

Move the read, parse, YAML-node mutation, section-plan composition, marshal, and `validateProspectiveChangedFields` work into `PlanSchemaInferences`. Return original bytes with `Modified: false` when no mutation applies. Copy action slices into `SchemaInferencePlan.Result` so callers cannot mutate shared backing arrays.

- [ ] **Step 4: Run planner tests and verify GREEN**

```bash
go test ./internal/infer -run 'TestPlanSchemaInferences' -count=1
```

Expected: PASS.

- [ ] **Step 5: Write atomic apply and compatibility-wrapper tests**

Add tests that seed a `0o600` `.stem`, plan a change, apply it, and assert content changed while mode remains `0o600`. Add a missing-parent failure by planning first, removing the parent directory, then calling `ApplySchemaInferencePlan` and asserting an error.

Add table-driven parity tests comparing the old public wrapper's result/actions for dry-run and real mode.

- [ ] **Step 6: Implement atomic persistence and wrapper delegation**

```go
func ApplySchemaInferencePlan(plan *SchemaInferencePlan) error {
    if plan == nil || !plan.Modified {
        return nil
    }
    if err := fsx.WriteFileAtomic(plan.Target, plan.Content, 0o644); err != nil {
        return fmt.Errorf("writing stem: %w", err)
    }
    return nil
}

func ApplySchemaInferences(stemPath string, inferences []ReportInference, dryRun bool) (*ApplyResult, error) {
    plan, err := PlanSchemaInferences(stemPath, inferences)
    if err != nil {
        return nil, err
    }
    result := plan.Result
    result.DryRun = dryRun
    if !dryRun {
        if err := ApplySchemaInferencePlan(plan); err != nil {
            return nil, err
        }
    }
    return &result, nil
}
```

Remove the direct `os.WriteFile(stemPath, ...)` branch from `apply.go`.

- [ ] **Step 7: Run infer and e2e tests**

```bash
go test ./internal/infer -race -count=1
go test ./internal/e2e -run 'Test.*SchemaApply' -race -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit Task 3**

```bash
git add internal/infer/apply.go internal/infer/apply_plan.go internal/infer/apply_plan_test.go internal/infer/apply_test.go internal/e2e/schema_apply_e2e_test.go
git commit -m "refactor(infer): separate schema planning from writes"
```

- [ ] **Step 9: Request task review**

Request a read-only review focused on public API compatibility, YAML preservation, no-write planning, and file-mode safety. Fix all Critical/Important findings.

---

### Task 4: Centralize prospective validation and atomic execution

**Files:**
- Create: `cmd/rootline/schema_apply_execution.go`
- Create: `cmd/rootline/schema_apply_execution_test.go`
- Modify: `cmd/rootline/schema.go`

**Interfaces:**
- Consumes: `rules.DiscoverStemState`, `rules.StemState.Overlay`, `rules.EvaluateStemState`, `fsx.WriteFileAtomic`.
- Produces:

```go
type stemWritePlan struct {
    reportTarget string
    target       string
    content      []byte
    action       string
}

type schemaApplyBatchPlan struct {
    writes          []stemWritePlan
    actionsByTarget map[string][]string
}

type stemWriteFunc func(string, []byte, fs.FileMode) error

func validateProspectiveStemWrites(ctx context.Context, root string, plan schemaApplyBatchPlan) ([]rules.StemHealthDiagnostic, error)
func executeStemWrites(ctx context.Context, plan schemaApplyBatchPlan, dryRun bool, write stemWriteFunc) (applied []string, errs []string)
func blockingStemHealth(diags []rules.StemHealthDiagnostic) []rules.StemHealthDiagnostic
```

- [ ] **Step 1: Write batch-wide validation tests**

Add direct helper tests for:

```go
func TestValidateProspectiveStemWritesRejectsJointlyInvalidHierarchy(t *testing.T)
func TestValidateProspectiveStemWritesAllowsWarningsAndInfo(t *testing.T)
func TestValidateProspectiveStemWritesUsesFinalDuplicateTargetContent(t *testing.T)
func TestValidateProspectiveStemWritesSortsDiagnostics(t *testing.T)
```

The first test must use a root enum and child string candidate and assert the returned diagnostics contain error-severity `type-consistency` owned by the child.

- [ ] **Step 2: Run prospective helper tests and verify RED**

```bash
go test ./cmd/rootline -run 'TestValidateProspectiveStemWrites' -count=1
```

Expected: compilation failure because the shared plan and validator do not exist.

- [ ] **Step 3: Implement virtual batch validation**

Sort writes stably by normalized target. Discover state once. Overlay every candidate in execution order so duplicate targets end at the exact final bytes. Evaluate once. Convert to `StemHealthDiagnostics`, sort them, and return them. Candidate parse failure returns a contextual planning error naming `reportTarget`; it must not mutate the baseline state.

`blockingStemHealth` filters only `SeverityError`.

- [ ] **Step 4: Write executor best-effort tests with an injected writer**

Use three targets and a writer function that fails only for the middle target:

```go
func TestExecuteStemWritesContinuesAfterAtomicWriteFailure(t *testing.T) {
    // expect actions for first and third, one error for second, and all three calls
}

func TestExecuteStemWritesStopsAfterContextCancellation(t *testing.T)
func TestExecuteStemWritesDryRunDoesNotCallWriter(t *testing.T)
func TestExecuteStemWritesUsesStableTargetOrder(t *testing.T)
```

- [ ] **Step 5: Run executor tests and verify RED**

```bash
go test ./cmd/rootline -run 'TestExecuteStemWrites' -count=1
```

Expected: compilation failure because the executor does not exist.

- [ ] **Step 6: Implement atomic best-effort execution**

For dry-run, append `actionsByTarget[target]` without calling `write`. For real execution, call the injected writer, append target actions only after success, collect contextual errors after failure, and continue. Before each operation, stop when `ctx.Err()` is non-nil and append one cancellation error.

Production callers pass `fsx.WriteFileAtomic`.

- [ ] **Step 7: Add the envelope field and shared initializer**

Modify `SchemaApplyResult`:

```go
StemHealth []rules.StemHealthDiagnostic `json:"stem_health"`
```

Initialize it as `[]rules.StemHealthDiagnostic{}` in both result constructors. Add a `newSchemaApplyResult(root string, dryRun bool)` helper to prevent the report paths from drifting.

- [ ] **Step 8: Run focused command tests**

```bash
go test ./cmd/rootline -run 'Test(ValidateProspectiveStemWrites|ExecuteStemWrites|SchemaApply.*Envelope)' -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit Task 4**

```bash
git add cmd/rootline/schema.go cmd/rootline/schema_apply_execution.go cmd/rootline/schema_apply_execution_test.go
git commit -m "refactor(schema): centralize validated stem execution"
```

- [ ] **Step 10: Request task review**

Request a read-only review focused on dry-run parity, duplicate targets, error severity, cancellation, and best-effort ordering. Fix all Critical/Important findings.

---

### Task 5: Gate schema-proposal writes on the complete virtual hierarchy

**Files:**
- Create: `cmd/rootline/schema_apply_hierarchy_test.go`
- Modify: `cmd/rootline/schema.go`
- Modify: `cmd/rootline/schema_apply_planner_test.go`
- Modify: `cmd/rootline/schema_apply_preflight_test.go`
- Modify: `cmd/rootline/schema_test.go`

**Interfaces:**
- Consumes: Task 4 `schemaApplyBatchPlan`, validator, executor, and result initializer.
- Produces: proposal reports converted to validated atomic write plans.

- [ ] **Step 1: Add the exact PR review regression**

Create a table over dry-run and real execution:

```go
func TestSchemaApplyProposalRejectsEnumToStringChildBeforePublication(t *testing.T) {
    // root .stem: estado enum [Pending, Done]
    // sub/record.md: estado Pending
    // proposal: create sub/.stem with estado string
    // expect non-zero, complete false, applied empty,
    // stem_health contains error type-consistency, target absent
}
```

Also assert `errors[0]` names `sub/.stem`, `type-consistency`, and `estado`.

- [ ] **Step 2: Run the regression and verify RED**

```bash
go test ./cmd/rootline -run 'TestSchemaApplyProposalRejectsEnumToStringChildBeforePublication' -count=1
```

Expected: FAIL because current apply returns exit 0 and creates the child stem in real mode.

- [ ] **Step 3: Convert proposal planning to the shared plan**

Replace `schemaProposalApplyPlan` with `stemWritePlan`. Preserve containment, skip/reject, overwrite guard, virtual target existence, byte-identical patch content, and resolved-target reporting.

Populate actions exactly:

```go
batch.actionsByTarget[target] = append(
    batch.actionsByTarget[target],
    fmt.Sprintf("%s: %s", action, target),
)
```

Keep stable duplicate-target order under `--force`.

- [ ] **Step 4: Insert the prospective gate before actions**

After proposal planning and before dry-run publication or execution:

1. call `validateProspectiveStemWrites`;
2. store all non-passing diagnostics in `result.StemHealth`;
3. convert each error-severity diagnostic to one deterministic `errors[]` string;
4. emit and return non-zero when any blocking diagnostic exists;
5. otherwise call `executeStemWrites`.

Remove direct `os.WriteFile(op.target, ...)` and the claim that isolated patch validation is the whole prospective check. Keep `runPostApplyValidation` only after real writes.

- [ ] **Step 5: Run the regression and planner suites**

```bash
go test ./cmd/rootline -run 'Test(SchemaApplyProposal|SchemaApply_.*Proposal|PlanSchemaProposalApply)' -count=1
```

Expected: PASS.

- [ ] **Step 6: Add joint-candidate and warning-only CLI tests**

Add one report where individually valid parent/child candidates conflict only when composed; assert the whole batch is rejected and no target exists. Add one report producing only a `scope-match` warning; assert exit 0, action published, and warning retained in `stem_health`.

- [ ] **Step 7: Run all schema proposal apply tests**

```bash
go test ./cmd/rootline -run 'TestSchemaApply.*(Proposal|Planner|Preflight|Hierarchy)' -race -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit Task 5**

```bash
git add cmd/rootline/schema.go cmd/rootline/schema_apply_hierarchy_test.go cmd/rootline/schema_apply_planner_test.go cmd/rootline/schema_apply_preflight_test.go cmd/rootline/schema_test.go
git commit -m "fix(schema): reject invalid prospective hierarchies"
```

- [ ] **Step 9: Request task review**

Request a read-only review against the original reproduction and proposal-path compatibility contracts. Fix all Critical/Important findings.

---

### Task 6: Route analyze reports through the same safety gate

**Files:**
- Modify: `cmd/rootline/schema.go`
- Modify: `cmd/rootline/schema_apply_hierarchy_test.go`
- Modify: `cmd/rootline/schema_apply_preflight_test.go`
- Modify: `cmd/rootline/schema_test.go`
- Modify: `internal/e2e/schema_apply_e2e_test.go`

**Interfaces:**
- Consumes: `infer.PlanSchemaInferences`, shared batch validation/execution, existing inference routing filter.
- Produces: analyze-derived candidate bytes that are validated before any action or write.

- [ ] **Step 1: Write analyze hierarchy regressions**

Add:

```go
func TestSchemaApplyAnalyzeRejectsProspectiveParentConflict(t *testing.T)
func TestSchemaApplyAnalyzeDryRunAndRealShareGovernanceVerdict(t *testing.T)
func TestSchemaApplyAnalyzePreservesModeAfterAtomicWrite(t *testing.T)
```

Use a closest child stem that inherits a stricter parent and an inference that produces a locally valid but inherited-invalid final field declaration. Snapshot target bytes and mode for both dry-run and rejection cases.

- [ ] **Step 2: Run analyze regressions and verify RED**

```bash
go test ./cmd/rootline -run 'TestSchemaApplyAnalyze.*(ProspectiveParentConflict|GovernanceVerdict|PreservesMode)' -count=1
```

Expected: at least the parent-conflict test fails because the current path writes through `ApplySchemaInferences` before hierarchy evaluation.

- [ ] **Step 3: Replace direct inference apply with planning**

In `runSchemaApplyFromAnalyze`:

1. keep report/root/preflight/closest-stem/inference-routing logic;
2. call `infer.PlanSchemaInferences(stemPath, schemaInferences)`;
3. stage `plan.Result.Skipped` and `Rejected` immediately;
4. if `plan.Modified`, create one `stemWritePlan` with `plan.Content`;
5. store `plan.Result.Applied` in `actionsByTarget[stemPath]` without copying them to `result.Applied` yet;
6. validate the virtual batch;
7. publish actions only through `executeStemWrites` after validation succeeds.

This preserves existing field-level action strings for analyze reports while sharing write safety.

- [ ] **Step 4: Run analyze regressions and verify GREEN**

```bash
go test ./cmd/rootline -run 'TestSchemaApplyAnalyze' -count=1
```

Expected: PASS.

- [ ] **Step 5: Prove both paths expose the same diagnostics**

Create equivalent proposal and analyze fixtures whose candidate child widens the same parent field. Compare `StemHealth` slices after normalizing only the report-specific root/target setup; severity, check, field, and relative path must match exactly.

- [ ] **Step 6: Run command and e2e suites**

```bash
go test ./cmd/rootline -run 'TestSchemaApply' -race -count=1
go test ./internal/e2e -run 'Test.*SchemaApply' -race -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit Task 6**

```bash
git add cmd/rootline/schema.go cmd/rootline/schema_apply_hierarchy_test.go cmd/rootline/schema_apply_preflight_test.go cmd/rootline/schema_test.go internal/e2e/schema_apply_e2e_test.go
git commit -m "fix(schema): validate analyze plans before writing"
```

- [ ] **Step 8: Request task review**

Request a read-only review focused on action timing, analyze/proposal parity, mode preservation, and absence of writes before governance validation. Fix all Critical/Important findings.

---

### Task 7: Publish the version-1 contract and run release gates

**Files:**
- Modify: `cmd/rootline/schema.go`
- Modify: `cmd/rootline/schema_test.go`
- Modify: `.claude/skills/rootline/ref-schema.md`
- Modify: `docs/analyze.md`
- Modify: `CHANGELOG.md`

**Interfaces:**
- Consumes: completed implementation from Tasks 1–6.
- Produces: documented version-1 `rootline/schema-apply` envelope and verified repository state.

- [ ] **Step 1: Add exact envelope contract tests**

For success with no diagnostics, assert marshaled JSON contains:

```json
"stem_health":[]
```

For a warning-only result, assert `complete:true`, exit 0, and a populated diagnostic. For a blocking result, assert `complete:false`, exit non-zero, `applied:[]`, and matching structured/string diagnostics. Add `TestSchemaApplyProspectiveHealthDoesNotReclassifyInvalidDocuments`: apply a governance-valid enum that an existing record violates and assert `complete:true`, empty `stem_health`, and a `validation_summary` with one invalid file. Add table-output assertions for a `Stem Health` section only when diagnostics are non-empty.

- [ ] **Step 2: Run contract tests and verify RED where output is incomplete**

```bash
go test ./cmd/rootline -run 'TestSchemaApply.*(Envelope|StemHealth|Table)' -count=1
```

Expected: failures until JSON initialization and table rendering are complete.

- [ ] **Step 3: Complete JSON and table rendering**

Keep `stem_health` non-omitempty and initialize it on every result path. Render rows with `Path`, `Check`, `Field`, `Severity`, and `Message`. Do not add an envelope version bump.

- [ ] **Step 4: Update living documentation**

In `.claude/skills/rootline/ref-schema.md`, document:

- complete virtual hierarchy validation before dry-run actions or writes;
- error-severity blocking versus non-blocking warnings/info;
- `stem_health[]` in the version-1 envelope;
- atomic per-file replacement and best-effort multi-file continuation;
- unchanged post-apply document `validation_summary` semantics.

In `docs/analyze.md`, state that analyze-derived changes are planned in memory and pass the same gate as schema proposals.

In `CHANGELOG.md`, add one unreleased bullet naming prospective hierarchy rejection and atomic `.stem` replacement.

- [ ] **Step 5: Run documentation contract tests**

```bash
go test ./cmd/rootline -run 'TestDocumentation|TestSchemaApply.*(Envelope|Table)' -count=1
```

Expected: PASS.

- [ ] **Step 6: Run formatting, lint, build, and tests**

```bash
just check
just test
```

Expected: zero formatting findings, zero lint issues, successful build, and all race-enabled tests PASS.

- [ ] **Step 7: Run coverage and vet gates**

```bash
just coverage-check
go vet ./...
```

Expected: every package at or above 85%, total coverage reported, and no vet findings.

- [ ] **Step 8: Validate governed documentation with the branch binary**

```bash
go run ./cmd/rootline validate --all docs/roadmap/ --output json --field summary
go run ./cmd/rootline validate docs/superpowers/specs/2026-08-20-prospective-schema-apply-safety-design.md --output json --field summary
go run ./cmd/rootline validate docs/superpowers/plans/2026-08-20-prospective-schema-apply-safety.md --output json --field summary
go run ./cmd/rootline validate --all docs/adr/ --output json --field summary
```

Expected: every summary reports zero invalid records, zero errors, and zero stem-health errors.

- [ ] **Step 9: Re-run the original failure as a focused regression**

```bash
go test ./cmd/rootline -run 'TestSchemaApplyProposalRejectsEnumToStringChildBeforePublication' -count=1
```

Expected: PASS for dry-run and real subtests, with no target written.

- [ ] **Step 10: Commit Task 7**

```bash
git add cmd/rootline/schema.go cmd/rootline/schema_test.go .claude/skills/rootline/ref-schema.md docs/analyze.md CHANGELOG.md
git commit -m "docs(schema): document prospective apply safety"
```

- [ ] **Step 11: Request final code review**

Request a final read-only review across the implementation range from the commit before Task 1 through Task 7. Fix all Critical and Important findings, rerun Steps 6–9 after any code change, and record Minor findings separately.

- [ ] **Step 12: Confirm branch state**

```bash
git status --short --branch
git log --oneline -9
```

Expected: clean worktree; the two approved design commits followed by seven implementation commits in task order.
