---
estado: Specified
---

# Schema Apply Residual Boundaries Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make prospective schema validation and atomic persistence operate on one stable physical target identity while preserving malformed external ancestors as structured stem-health diagnostics.

**Architecture:** `internal/fsx` will bind each accepted write to an opened physical parent capability. Schema apply will coalesce, overlay, and execute those physical identities while retaining lexical paths only for reporting. `internal/rules` will replace fail-fast external `WalkUp` discovery with a tolerant collector that records parse failures and continues to the nearest valid root marker.

**Tech Stack:** Go 1.26+, `os.Root`, `os.SameFile`, standard `testing`, Cobra command tests, Rootline `StemState` and stem-health evaluator.

**Spec:** `docs/superpowers/specs/2026-08-20-schema-apply-residual-boundaries-design.md`

## Global Constraints

- Work only in `/Users/Shared/harness/.worktrees/rootline-schema-contract-convergence`.
- Preserve schema-apply envelope version 1 and always-initialized `stem_health`.
- Preserve dry-run/write validation parity.
- Preserve atomic replacement per file and best-effort execution across files.
- Continue later independent writes after one write failure unless context is canceled.
- Preserve existing modes when replacing `.stem`; use `0644` only for creation.
- Support internal parent-directory symlinks; reject physical escape.
- External malformed ancestor diagnostics use report-root-relative paths such as `../.stem`.
- Do not implement the general injectable filesystem enhancement tracked by #194.
- Do not push or merge.

---

## File Structure

- `internal/fsx/atomic_target.go` — owns physical target resolution, opened parent identity, atomic target-local replacement, and capability cleanup.
- `internal/fsx/atomic_target_test.go` — directly proves internal symlink support, escape rejection, retarget stability, parent substitution detection, mode preservation, and cleanup.
- `internal/rules/stem_state.go` — replaces external `WalkUp` use with tolerant ancestor collection into `StemState`.
- `internal/rules/stem_state_test.go` — proves valid and malformed external ancestor collection, root-marker traversal, and cancellation/error behavior.
- `cmd/rootline/schema_apply_execution.go` — binds plans to capabilities, coalesces by physical identity, overlays physical paths, executes through capabilities, and closes them.
- `cmd/rootline/schema_apply_execution_test.go` — tests capability-bound ordering, alias coalescing, retarget stability, failure publication, and cleanup.
- `cmd/rootline/schema.go` — makes proposal and analyze planning resolve capabilities before overwrite/existence policy is finalized.
- `cmd/rootline/schema_apply_planner_test.go` — adapts planner contracts and covers containment/overwrite classification through bound targets.
- `cmd/rootline/schema_apply_hierarchy_test.go` — adds proposal/analyze command regressions for physical alias hierarchy and malformed external ancestors.
- `.claude/skills/rootline/ref-schema.md` — documents physical-target convergence and structured external governance failures.
- `docs/analyze.md` — documents analyze behavior when the governing `.stem` is external or malformed.
- `CHANGELOG.md` — records the residual safety correction in the unreleased section.

---

### Task 1: Add the opened physical target capability

**Files:**
- Create: `internal/fsx/atomic_target.go`
- Create: `internal/fsx/atomic_target_test.go`
- Reuse: `internal/fsx/atomic.go:131-174`

**Interfaces:**
- Consumes: `writeFileAtomicRoot(root *os.Root, relTarget string, perm fs.FileMode, write func(io.Writer) error) error`.
- Produces:
  ```go
  func ResolveAtomicTarget(allowedRoot, logicalTarget string) (*AtomicTarget, error)
  func (t *AtomicTarget) LogicalPath() string
  func (t *AtomicTarget) PhysicalPath() string
  func (t *AtomicTarget) Stat() (fs.FileInfo, error)
  func (t *AtomicTarget) WriteFileAtomic(content []byte, perm fs.FileMode) error
  func (t *AtomicTarget) Close() error
  ```

- [ ] **Step 1: Write failing tests for internal symlink resolution and retarget stability**

Create `internal/fsx/atomic_target_test.go` with Unix-only symlink fixtures:

```go
func TestAtomicTargetFollowsInternalAliasOnce(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("symlink setup requires Unix-compatible semantics")
    }
    root := t.TempDir()
    original := filepath.Join(root, "original")
    redirected := filepath.Join(root, "redirected")
    for _, dir := range []string{original, redirected} {
        if err := os.MkdirAll(dir, 0o755); err != nil { t.Fatal(err) }
    }
    alias := filepath.Join(root, "alias")
    if err := os.Symlink("original", alias); err != nil { t.Fatal(err) }

    target, err := ResolveAtomicTarget(root, filepath.Join(alias, ".stem"))
    if err != nil { t.Fatal(err) }
    defer func() { _ = target.Close() }()
    if got, want := target.PhysicalPath(), filepath.Join(original, ".stem"); got != want {
        t.Fatalf("PhysicalPath() = %q, want %q", got, want)
    }

    if err := os.Remove(alias); err != nil { t.Fatal(err) }
    if err := os.Symlink("redirected", alias); err != nil { t.Fatal(err) }
    if err := target.WriteFileAtomic([]byte("version: 2\n"), 0o644); err != nil { t.Fatal(err) }
    if _, err := os.Stat(filepath.Join(redirected, ".stem")); !errors.Is(err, fs.ErrNotExist) {
        t.Fatalf("redirected target exists: %v", err)
    }
    if got, err := os.ReadFile(filepath.Join(original, ".stem")); err != nil || string(got) != "version: 2\n" {
        t.Fatalf("original target = %q, %v", got, err)
    }
}
```

Add `TestResolveAtomicTargetRejectsPhysicalEscape`, using `root/alias -> ../outside` and asserting the outside `.stem` is never created.

- [ ] **Step 2: Run the focused tests and verify RED**

Run:

```bash
go test ./internal/fsx -run 'TestAtomicTarget|TestResolveAtomicTarget' -count=1
```

Expected: compilation fails because `ResolveAtomicTarget` and `AtomicTarget` do not exist.

- [ ] **Step 3: Implement physical resolution and identity verification**

Create `internal/fsx/atomic_target.go`:

```go
package fsx

import (
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
    "sync"
)

type AtomicTarget struct {
    logicalPath  string
    physicalPath string
    parentPath   string
    parent       *os.Root
    name         string
    closeOnce    sync.Once
    closeErr     error
}

func ResolveAtomicTarget(allowedRoot, logicalTarget string) (*AtomicTarget, error) {
    logicalTarget, err := filepath.Abs(logicalTarget)
    if err != nil { return nil, fmt.Errorf("resolving target %s: %w", logicalTarget, err) }
    physicalRoot, err := filepath.EvalSymlinks(allowedRoot)
    if err != nil { return nil, fmt.Errorf("resolving allowed root %s: %w", allowedRoot, err) }
    physicalRoot, err = filepath.Abs(physicalRoot)
    if err != nil { return nil, fmt.Errorf("normalizing allowed root %s: %w", allowedRoot, err) }
    physicalParent, err := filepath.EvalSymlinks(filepath.Dir(logicalTarget))
    if err != nil { return nil, fmt.Errorf("resolving parent for %s: %w", logicalTarget, err) }
    physicalParent, err = filepath.Abs(physicalParent)
    if err != nil { return nil, fmt.Errorf("normalizing parent for %s: %w", logicalTarget, err) }
    if !pathAtOrBelow(physicalRoot, physicalParent) {
        return nil, fmt.Errorf("target %s escapes root %s", logicalTarget, allowedRoot)
    }
    parent, err := os.OpenRoot(physicalParent)
    if err != nil { return nil, fmt.Errorf("opening physical parent %s: %w", physicalParent, err) }
    target := &AtomicTarget{
        logicalPath: filepath.Clean(logicalTarget),
        physicalPath: filepath.Join(filepath.Clean(physicalParent), filepath.Base(logicalTarget)),
        parentPath: filepath.Clean(physicalParent),
        parent: parent,
        name: filepath.Base(logicalTarget),
    }
    if err := target.verifyParentIdentity(); err != nil {
        _ = parent.Close()
        return nil, err
    }
    return target, nil
}

func pathAtOrBelow(root, path string) bool {
    rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
    return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (t *AtomicTarget) verifyParentIdentity() error {
    opened, err := t.parent.Stat(".")
    if err != nil { return fmt.Errorf("stating opened parent for %s: %w", t.logicalPath, err) }
    named, err := os.Stat(t.parentPath)
    if err != nil { return fmt.Errorf("stating physical parent for %s: %w", t.logicalPath, err) }
    if !os.SameFile(opened, named) {
        return fmt.Errorf("physical parent changed for %s", t.logicalPath)
    }
    return nil
}

func (t *AtomicTarget) LogicalPath() string { return t.logicalPath }
func (t *AtomicTarget) PhysicalPath() string { return t.physicalPath }
func (t *AtomicTarget) Stat() (fs.FileInfo, error) { return t.parent.Stat(t.name) }
func (t *AtomicTarget) WriteFileAtomic(content []byte, perm fs.FileMode) error {
    if err := t.verifyParentIdentity(); err != nil { return err }
    return writeFileAtomicRoot(t.parent, t.name, perm, func(dst io.Writer) error {
        _, err := dst.Write(content)
        return err
    })
}
func (t *AtomicTarget) Close() error {
    t.closeOnce.Do(func() { t.closeErr = t.parent.Close() })
    return t.closeErr
}
```

Include the missing `io` import. Keep `pathAtOrBelow` private to `fsx`; do not introduce a filesystem interface.

- [ ] **Step 4: Add identity-substitution, mode, stat, and cleanup tests**

Add tests with these exact assertions:

```go
func TestAtomicTargetRejectsReplacedPhysicalParent(t *testing.T) {
    root := t.TempDir()
    parent := filepath.Join(root, "parent")
    if err := os.Mkdir(parent, 0o755); err != nil { t.Fatal(err) }
    target, err := ResolveAtomicTarget(root, filepath.Join(parent, ".stem"))
    if err != nil { t.Fatal(err) }
    defer func() { _ = target.Close() }()
    moved := filepath.Join(root, "moved")
    if err := os.Rename(parent, moved); err != nil { t.Fatal(err) }
    if err := os.Mkdir(parent, 0o755); err != nil { t.Fatal(err) }
    if err := target.WriteFileAtomic([]byte("new"), 0o644); err == nil || !strings.Contains(err.Error(), "physical parent changed") {
        t.Fatalf("WriteFileAtomic error = %v", err)
    }
    for _, path := range []string{filepath.Join(parent, ".stem"), filepath.Join(moved, ".stem")} {
        if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) { t.Fatalf("unexpected write at %s: %v", path, err) }
    }
}
```

Also assert:

- `Stat()` returns `fs.ErrNotExist` for a create candidate;
- replacing an existing `0600` target preserves `0600`;
- `Close()` may be called twice without panicking or closing an unrelated handle;
- a partial-write injection against `writeFileAtomicRoot` still leaves no staging debris (retain existing test coverage).

- [ ] **Step 5: Run package tests and coverage**

Run:

```bash
go test ./internal/fsx -race -count=1
go test ./internal/fsx -cover
```

Expected: PASS and package coverage at or above 85%.

- [ ] **Step 6: Commit the capability**

```bash
git add internal/fsx/atomic_target.go internal/fsx/atomic_target_test.go
git commit -m "fix(fsx): bind atomic writes to physical targets"
```

---

### Task 2: Preserve malformed external ancestors in StemState

**Files:**
- Modify: `internal/rules/stem_state.go:33-111`
- Modify: `internal/rules/stem_state_test.go:93-136`
- Test: `internal/rules/stem_state_health_test.go`

**Interfaces:**
- Consumes: `addStemToState(state *StemState, path string, content []byte)` and `StemFile.Root`.
- Produces: private `collectExternalStemState(ctx context.Context, state *StemState, startDir string) error`.

- [ ] **Step 1: Write failing discovery tests for malformed external governance**

Add to `internal/rules/stem_state_test.go`:

```go
func TestDiscoverStemStatePreservesMalformedExternalAncestorAndContinuesToRoot(t *testing.T) {
    grand := t.TempDir()
    parent := filepath.Join(grand, "parent")
    root := filepath.Join(parent, "child")
    mustWriteStemStateFile(t, filepath.Join(grand, ".stem"), "version: 2\nroot: true\n")
    malformed := filepath.Join(parent, ".stem")
    mustWriteStemStateFile(t, malformed, "version: [broken\n")
    mustWriteStemStateFile(t, filepath.Join(root, "record.md"), "# Record\n")

    state, err := DiscoverStemState(context.Background(), root)
    if err != nil { t.Fatal(err) }
    if state.ParseErrors[malformed] == nil { t.Fatal("external parse error was not retained") }
    if state.Stems[filepath.Join(grand, ".stem")] == nil { t.Fatal("collector did not continue to valid root marker") }
    if containsStemStatePath(state.EvaluatedStemPaths(), malformed) {
        t.Fatal("untouched external malformed ancestor became scan-owned")
    }
}
```

Add a health test that promotes the malformed external path through an overlay candidate and asserts `StemHealthDiagnostics` includes `{Path: "../.stem", Check: "yaml-valid", Severity: "error"}`.

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
go test ./internal/rules -run 'TestDiscoverStemStatePreservesMalformedExternal|TestEvaluateStemState.*External' -count=1
```

Expected: FAIL because `WalkUp` returns the parse error before `StemState.ParseErrors` is populated.

- [ ] **Step 3: Replace external WalkUp with a tolerant collector**

In `internal/rules/stem_state.go`, replace the `WalkUp(filepath.Dir(absRoot))` block with:

```go
if err := collectExternalStemState(ctx, state, filepath.Dir(absRoot)); err != nil {
    return nil, err
}
```

Implement:

```go
func collectExternalStemState(ctx context.Context, state *StemState, startDir string) error {
    for dir := filepath.Clean(startDir); ; dir = filepath.Dir(dir) {
        if err := ctx.Err(); err != nil { return err }
        stemPath := filepath.Join(dir, stemFileName)
        content, err := os.ReadFile(stemPath)
        switch {
        case err == nil:
            stem, parseErr := ParseStem(stemPath, content)
            if parseErr != nil {
                state.ParseErrors[stemPath] = parseErr
                delete(state.Stems, stemPath)
            } else {
                state.Stems[stemPath] = stem
                delete(state.ParseErrors, stemPath)
                if stem.Root { return nil }
            }
        case errors.Is(err, fs.ErrNotExist):
        default:
            return fmt.Errorf("reading external stem %s: %w", stemPath, err)
        }
        parent := filepath.Dir(dir)
        if parent == dir { return nil }
    }
}
```

Add `io/fs` to imports. Do not modify shared `WalkUp`; other consumers retain existing behavior.

Extend `StemState.Overlay` so a candidate promotes malformed stems in its governing chain into `Evaluated` ownership:

```go
func (s *StemState) markCandidateChainEvaluated(path string) {
    for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
        stemPath := filepath.Join(dir, stemFileName)
        if _, malformed := s.ParseErrors[stemPath]; malformed {
            s.Evaluated[stemPath] = true
        }
        if stem := s.Stems[stemPath]; stem != nil && stem.Root {
            return
        }
        parent := filepath.Dir(dir)
        if parent == dir { return }
    }
}
```

Call `clone.markCandidateChainEvaluated(absPath)` after marking the candidate itself. This keeps untouched external diagnostics context-only while ensuring every malformed ancestor governing an explicit write is evaluated.

- [ ] **Step 4: Add cancellation and operational-read-error tests**

- Use an already-canceled context and assert `errors.Is(err, context.Canceled)`.
- Create an external `.stem` as a directory and assert `os.ReadFile` produces an operational error rather than populating `ParseErrors`.
- Assert a malformed external stem cannot stop discovery even if its bytes contain the text `root: true` outside valid YAML.

- [ ] **Step 5: Run rules tests and coverage**

```bash
go test ./internal/rules -run 'TestDiscoverStemState|TestStemState|TestEvaluateStemState' -race -count=1
go test ./internal/rules -cover
```

Expected: PASS and coverage at or above 85%.

- [ ] **Step 6: Commit tolerant discovery**

```bash
git add internal/rules/stem_state.go internal/rules/stem_state_test.go internal/rules/stem_state_health_test.go
git commit -m "fix(rules): retain malformed external stems"
```

---

### Task 3: Bind schema plans, validation, and execution to capabilities

**Files:**
- Modify: `cmd/rootline/schema_apply_execution.go:17-150`
- Modify: `cmd/rootline/schema.go:417-496`
- Modify: `cmd/rootline/schema.go:542-710`
- Modify: `cmd/rootline/schema_apply_execution_test.go`
- Modify: `cmd/rootline/schema_apply_planner_test.go`

**Interfaces:**
- Consumes from Task 1: `fsx.ResolveAtomicTarget`, `AtomicTarget.PhysicalPath`, `AtomicTarget.Stat`, `AtomicTarget.WriteFileAtomic`, and `AtomicTarget.Close`.
- Consumes from Task 2: tolerant `rules.DiscoverStemState`.
- Produces:
  ```go
  type stemWritePlan struct {
      reportTarget string
      targetPath   string
      target       *fsx.AtomicTarget
      content      []byte
      action       string
  }
  func closeSchemaApplyBatch(plan schemaApplyBatchPlan) error
  ```

- [ ] **Step 1: Write failing executor tests for physical identity and alias coalescing**

Adapt `cmd/rootline/schema_apply_execution_test.go` so real target plans use a helper:

```go
func mustAtomicStemTarget(t *testing.T, root, logical string) *fsx.AtomicTarget {
    t.Helper()
    target, err := fsx.ResolveAtomicTarget(root, logical)
    if err != nil { t.Fatal(err) }
    t.Cleanup(func() { _ = target.Close() })
    return target
}
```

Add:

```go
func TestSortedSchemaApplyBatchCoalescesAliasesByPhysicalTarget(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("symlink semantics differ") }
    root := t.TempDir()
    physical := filepath.Join(root, "physical")
    if err := os.Mkdir(physical, 0o755); err != nil { t.Fatal(err) }
    for _, alias := range []string{"one", "two"} {
        if err := os.Symlink("physical", filepath.Join(root, alias)); err != nil { t.Fatal(err) }
    }
    first := mustAtomicStemTarget(t, root, filepath.Join(root, "one", ".stem"))
    second := mustAtomicStemTarget(t, root, filepath.Join(root, "two", ".stem"))
    plan := schemaApplyBatchPlan{
        writes: []stemWritePlan{
            {reportTarget: "one/.stem", target: first, content: []byte("first")},
            {reportTarget: "two/.stem", target: second, content: []byte("final")},
        },
        actionsByWrite: [][]string{{"first action"}, {"second action"}},
    }
    items := sortedSchemaApplyBatch(plan)
    if len(items) != 1 || string(items[0].write.content) != "final" {
        t.Fatalf("items = %+v", items)
    }
    if !reflect.DeepEqual(items[0].actions, []string{"first action", "second action"}) {
        t.Fatalf("actions = %#v", items[0].actions)
    }
}
```

Add a validation fixture where `root/link` points to `root/a/deep`, the lexical hierarchy would pass, the physical hierarchy conflicts, and `validateProspectiveStemWrites` must emit the physical target's `type-consistency` error.

- [ ] **Step 2: Run focused tests and verify RED**

```bash
go test ./cmd/rootline -run 'TestSortedSchemaApplyBatchCoalescesAliases|TestValidateProspectiveStemWrites.*Physical' -count=1
```

Expected: compile failures because `stemWritePlan.target` is still a string and grouping uses lexical paths.

- [ ] **Step 3: Change write plans to hold bound capabilities**

Change `stemWritePlan` to:

```go
type stemWritePlan struct {
    reportTarget string
    targetPath   string
    target       *fsx.AtomicTarget
    content      []byte
    action       string
}
```

Use `targetPath` only while constructing or reporting the plan. Update `normalizedStemWriteTarget` to accept a plan and return `write.target.PhysicalPath()`. Remove `writeRoot` from production plans.

Implement cleanup:

```go
func closeSchemaApplyBatch(plan schemaApplyBatchPlan) error {
    seen := map[*fsx.AtomicTarget]struct{}{}
    var errs []error
    for _, write := range plan.writes {
        if write.target == nil { continue }
        if _, ok := seen[write.target]; ok { continue }
        seen[write.target] = struct{}{}
        if err := write.target.Close(); err != nil { errs = append(errs, err) }
    }
    return errors.Join(errs...)
}
```

Every command path must install this fallback immediately after obtaining a non-empty plan:

```go
defer func() { _ = closeSchemaApplyBatch(plan) }()
```

Rejected capabilities are closed immediately by the planner. The deferred fallback owns accepted capabilities on every return path. As with existing `os.Root` cleanup, a close-only error does not rewrite an already computed apply envelope and never publishes actions.

- [ ] **Step 4: Make prospective validation overlay physical paths**

Replace rooted stat and lexical overlay logic with:

```go
for _, item := range sortedSchemaApplyBatch(plan) {
    if item.write.target == nil {
        return nil, fmt.Errorf("validating target %s: target capability unavailable", item.write.reportTarget)
    }
    next, err := state.Overlay(item.write.target.PhysicalPath(), item.write.content)
    if err != nil {
        return nil, fmt.Errorf("planning proposed .stem %s: %w", item.write.reportTarget, err)
    }
    state = next
}
```

Retain deterministic diagnostic sorting. Physical paths are internal state keys; public planning errors continue to name `reportTarget`.

- [ ] **Step 5: Execute through the capability and preserve test injection**

Replace `stemWriteFunc` with:

```go
type stemWriteFunc func(*fsx.AtomicTarget, []byte, fs.FileMode) error
```

Production passes:

```go
func(target *fsx.AtomicTarget, content []byte, mode fs.FileMode) error {
    return target.WriteFileAtomic(content, mode)
}
```

The executor calls the writer once per coalesced physical target and publishes all grouped actions only after success. Keep dry-run writer-free, cancellation behavior, stable physical-path ordering, and best-effort continuation.

- [ ] **Step 6: Resolve proposal capabilities before existence/overwrite classification**

Replace the stat-only planner dependency with:

```go
type schemaApplyTargetResolver func(root, target string) (*fsx.AtomicTarget, error)
```

`planSchemaProposalApplyWithResolver` must:

1. preserve existing lexical containment and report-target calculation;
2. call `fsx.ResolveAtomicTarget(scanRoot, target)`;
3. classify `target.Stat()` as existing, absent, or operational failure;
4. close rejected/skipped capabilities immediately;
5. retain accepted capabilities in `stemWritePlan`;
6. apply `force` using the capability's exact target identity.

Update planner tests to inject a resolver returning real temporary capabilities. Keep assertions for `rejected`, `skipped`, containment, overwrite refusal, and action order.

- [ ] **Step 7: Resolve analyze capabilities using the explicit write anchor**

After `rules.Resolve` selects `stemPath` and `infer.PlanSchemaInferences` reports `Modified`, choose the anchor with an exact local relative-path check:

```go
writeAnchor := root
rel, relErr := filepath.Rel(filepath.Clean(root), filepath.Clean(stemPath))
external := relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
if external {
    writeAnchor = filepath.Dir(stemPath)
}
target, err := fsx.ResolveAtomicTarget(writeAnchor, stemPath)
```

Store `target` in the batch and defer cleanup before validation. Do not change `internal/fix.ContainPath`; it remains the lexical policy classifier, while `AtomicTarget` is the physical authority.

- [ ] **Step 8: Run focused executor/planner tests**

```bash
go test ./cmd/rootline -run 'Test(ValidateProspectiveStemWrites|ExecuteStemWrites|SortedSchemaApplyBatch|PlanSchemaProposalApply)' -race -count=1
```

Expected: PASS. Confirm failure messages retain lexical `reportTarget` values.

- [ ] **Step 9: Commit shared integration**

```bash
git add cmd/rootline/schema_apply_execution.go cmd/rootline/schema.go cmd/rootline/schema_apply_execution_test.go cmd/rootline/schema_apply_planner_test.go
git commit -m "fix(schema): execute validated physical targets"
```

---

### Task 4: Add public regressions for both report paths

**Files:**
- Modify: `cmd/rootline/schema_apply_hierarchy_test.go`
- Modify: `cmd/rootline/schema_test.go`
- Modify: `internal/e2e/schema_apply_e2e_test.go`
- Modify: `.claude/skills/rootline/ref-schema.md`
- Modify: `docs/analyze.md`
- Modify: `CHANGELOG.md`

**Interfaces:**
- Consumes: complete capability-bound schema apply flow from Task 3.
- Produces: version 1 CLI contract coverage for physical aliases and external `yaml-valid` diagnostics.

- [ ] **Step 1: Add a proposal regression for lexical/physical hierarchy divergence**

In `cmd/rootline/schema_apply_hierarchy_test.go`, create a Unix-only fixture:

```text
root/.stem                    root: true
root/a/.stem                  estado: enum [Pending, Done]
root/a/deep/                  physical candidate parent
root/link -> a/deep           lexical report target parent
proposal candidate            estado: string at root/link/.stem
```

Invoke proposal apply in dry-run and real modes. Assert both:

- return non-zero;
- `complete == false`;
- `applied` is empty;
- `stem_health` contains `type-consistency`, field `estado`, severity `error`;
- neither `root/a/deep/.stem` nor another alias target is created or replaced.

The test name must be `TestSchemaApplyProposalValidatesPhysicalSymlinkTarget`.

- [ ] **Step 2: Add proposal and analyze regressions for malformed external ancestors**

Create a table test with report kinds `rootline/schema-proposals` and `rootline/analyze` using:

```text
project/.stem                 valid root: true
project/subtree/.stem         malformed YAML
project/subtree/work/         report root
```

For each report kind and dry-run boolean, assert:

```go
if got.Path != filepath.Join("..", ".stem") || got.Check != "yaml-valid" || got.Severity != rules.SeverityError {
    t.Fatalf("stem health = %+v", result.StemHealth)
}
if len(result.Applied) != 0 || result.Complete { t.Fatalf("result = %+v", result) }
```

Snapshot target bytes before invocation and assert byte identity afterward.

- [ ] **Step 3: Add an executor retarget regression**

Use the production `fsx.AtomicTarget` with an injected writer hook that retargets the lexical alias immediately before invoking `target.WriteFileAtomic`. Assert the original physical target receives the content and the redirected directory remains untouched. This proves the command executor does not reopen the lexical path.

- [ ] **Step 4: Add E2E parity coverage**

In `internal/e2e/schema_apply_e2e_test.go`, add one proposal dry-run/real pair for malformed external governance. Decode stdout before checking the non-zero exit and compare normalized `stem_health` keys between modes. Do not compare post-write document-validation fields.

- [ ] **Step 5: Run public contract tests**

```bash
go test ./cmd/rootline -run 'TestSchemaApply.*(PhysicalSymlink|MalformedExternal|Retarget)' -race -count=1
go test ./internal/e2e -run 'Test.*SchemaApply.*MalformedExternal' -race -count=1
```

Expected: PASS with identical blocking diagnostics across report kinds and modes.

- [ ] **Step 6: Update living documentation**

Add these explicit statements:

- `.claude/skills/rootline/ref-schema.md`: accepted writes are validated and replaced by one bound physical target; internal aliases are supported but escaping aliases are rejected.
- `docs/analyze.md`: a malformed governing `.stem` above the report root appears in `stem_health` with a relative path and blocks both dry-run and write.
- `CHANGELOG.md`: under Unreleased, note physical-target convergence and structured external ancestor diagnostics.

Do not add another ADR; ADR 0001 already records the decision.

- [ ] **Step 7: Validate docs and commit public coverage**

```bash
go run ./cmd/rootline validate docs/analyze.md --output json
go run ./cmd/rootline validate docs/adr/0001-evaluar-estado-prospectivo-de-stems.md --output json
git diff --check
git add cmd/rootline/schema_apply_hierarchy_test.go cmd/rootline/schema_test.go internal/e2e/schema_apply_e2e_test.go .claude/skills/rootline/ref-schema.md docs/analyze.md CHANGELOG.md
git commit -m "test(schema): cover residual apply boundaries"
```

Expected: both validation envelopes report zero invalid records and zero stem-health errors.

---

### Task 5: Run release gates and perform final architecture review

**Files:**
- Create: `.superpowers/sdd/2026-08-20-schema-apply-residual-boundaries/final-report.md`
- Modify only if a gate exposes a defect: files already listed in Tasks 1–4.

**Interfaces:**
- Consumes: all prior task commits.
- Produces: merge-readiness evidence; no push or merge.

- [ ] **Step 1: Run affected race suites**

```bash
go test ./internal/fsx ./internal/rules ./cmd/rootline ./internal/e2e -race -count=1
```

Expected: PASS.

- [ ] **Step 2: Run repository quality gates**

```bash
just check
just test
just coverage-check
go vet ./...
```

Expected: every command exits 0; every package remains at or above its configured 85% floor.

- [ ] **Step 3: Validate governed documentation with the branch binary**

```bash
go run ./cmd/rootline validate --all docs/roadmap/ --output json > /tmp/rootline-roadmap-validate.json
go run ./cmd/rootline validate --all docs/ --output json > /tmp/rootline-docs-validate.json
```

Inspect both envelopes and require:

- `summary.invalid == 0`;
- `summary.errors_count == 0`;
- `summary.stem_health_errors_count == 0`.

- [ ] **Step 4: Re-run the original PR #193 regression**

```bash
go test ./cmd/rootline -run 'TestSchemaApplyProposalRejectsEnumToStringChildBeforePublication' -race -count=1
```

Expected: PASS.

- [ ] **Step 5: Check repository integrity**

```bash
git diff --check
git status --short
git log --oneline -6
```

Expected: no uncommitted implementation files except the final report before it is committed; no files outside the assigned worktree changed.

- [ ] **Step 6: Write the final report**

Create `.superpowers/sdd/2026-08-20-schema-apply-residual-boundaries/final-report.md` containing:

- RED/GREEN evidence per task;
- exact gate commands and exit status;
- per-package and total coverage output;
- physical symlink retarget evidence;
- proposal/analyze malformed-ancestor parity evidence;
- original regression evidence;
- residual risks, explicitly including the unchanged last-writer content race and best-effort multi-file behavior;
- confirmation that #194 was not implemented.

- [ ] **Step 7: Request a read-only architecture review**

The reviewer must inspect, at minimum:

1. validation keys and execution target the same `AtomicTarget`;
2. aliases coalesce by `PhysicalPath`;
3. parent identity is checked immediately before staging;
4. malformed external discovery continues to a valid root marker;
5. operational I/O errors are not emitted as `yaml-valid`;
6. every capability closes on every command return path;
7. no version bump or filesystem-wide interface was introduced.

If the reviewer finds an Important or Critical issue, leave the task in progress and create a focused fix task rather than claiming completion.

- [ ] **Step 8: Commit the verification report after review passes**

```bash
git add .superpowers/sdd/2026-08-20-schema-apply-residual-boundaries/final-report.md
git commit -m "test(schema): verify residual apply safety"
```

Do not push or merge.
