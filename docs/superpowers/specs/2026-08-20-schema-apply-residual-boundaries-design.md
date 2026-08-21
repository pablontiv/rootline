---
estado: Specified
---

# Schema Apply Residual Boundary Safety Design

**Date:** 2026-08-20
**Status:** Approved design amendment
**Parent design:** `2026-08-20-prospective-schema-apply-safety-design.md`
**Follow-up enhancement:** #194

## Purpose

Close the two residual safety defects in prospective `schema apply` without broadening the change into a general filesystem abstraction:

1. validation can overlay a lexical path reached through an internal directory symlink while execution replaces a different physical `.stem`; and
2. malformed governing ancestors above the report root can fail through fail-fast walk-up resolution before `schema apply` emits a structured `yaml-valid` stem-health diagnostic.

The amendment preserves version 1 output envelopes, dry-run/write parity, atomic replacement per file, best-effort execution across files, and support for internal symlinks.

## Decisions

1. Every accepted write is bound during planning to one opened physical parent-directory capability.
2. Prospective validation overlays the physical target represented by that capability, not the lexical alias from the report.
3. Execution stages and renames through the same opened capability; it does not traverse the lexical path again.
4. Parent identity is checked when the capability is created and again before mutation. A moved or substituted physical parent fails safely before staging.
5. Duplicate writes are coalesced by physical target identity, so two lexical aliases cannot cause two physical replacements.
6. Public errors and actions retain the lexical report target. Stem-health paths remain relative to the report root.
7. External ancestor discovery becomes tolerant: malformed YAML is retained in `StemState.ParseErrors` and discovery continues until a valid `root: true` or the filesystem root.
8. Read errors other than absence remain operational errors; they are not mislabeled as YAML failures.
9. A general injectable or writable filesystem remains deferred to #194.

## Architecture

### Opened atomic target

`internal/fsx` owns a narrow capability for one target file:

```go
type AtomicTarget struct {
    logicalPath  string
    physicalPath string
    parentPath   string
    parent       *os.Root
    name         string
}
```

The concrete fields may remain private. The supported operations are conceptually:

```go
func ResolveAtomicTarget(allowedRoot, logicalTarget string) (*AtomicTarget, error)
func (t *AtomicTarget) PhysicalPath() string
func (t *AtomicTarget) WriteFileAtomic(content []byte, perm fs.FileMode) error
func (t *AtomicTarget) Close() error
```

`ResolveAtomicTarget` performs these steps:

1. normalize and resolve the physical `allowedRoot`;
2. resolve symlinks in the target parent;
3. confirm that the physical parent is at or below the permitted physical root;
4. open the physical parent with `os.OpenRoot`;
5. compare `parent.Stat(".")` with `os.Stat(parentPath)` using `os.SameFile`;
6. retain the opened handle, physical path, and basename.

Opening the already-resolved physical parent prevents a concurrent retarget of the lexical alias from redirecting the capability. The identity comparison detects substitution during resolution. Before staging, `WriteFileAtomic` repeats the handle-versus-path identity check. If the physical directory was moved, removed, or replaced, execution fails before modifying the target.

The capability follows internal symlinks but rejects physical escape. It does not prohibit mount traversal beyond the guarantees provided by `os.Root`; that platform behavior is unchanged and outside this amendment.

### Plan ownership

A planned write carries its public target, final content, actions, and capability:

```go
type stemWritePlan struct {
    reportTarget string
    target       *fsx.AtomicTarget
    content      []byte
    action       string
}
```

Proposal targets use the resolved report root as their allowed root. Analyze may legitimately select the closest governing `.stem` above the report root; that target is first selected through the existing trusted resolver, then its physical parent becomes the permitted anchor for that explicit external write.

The batch owns every capability from successful planning until validation and execution finish. Each capability is closed exactly once on every exit path, including parse failure, blocking stem health, dry-run, cancellation, and write failure.

Batch normalization groups writes by normalized physical target. The final candidate bytes win under the existing duplicate-target policy. All approved actions for the physical target are published only after its single physical write succeeds, or immediately after validation in dry-run mode.

### Physical prospective overlay

`validateProspectiveStemWrites` discovers the report-root state and overlays each unique write using `AtomicTarget.PhysicalPath()`. Therefore `StemState` evaluates the same `.stem` that execution can replace.

The logical report target is never used as a state key. It remains presentation metadata only. This separation prevents an internal symlink alias from creating a false virtual child while the executor mutates another governed directory.

A physical candidate remains evaluation-owned even when it lies above the report root. Untouched external ancestors remain context-only.

### Tolerant external ancestor collection

`DiscoverStemState` no longer delegates external discovery to fail-fast `WalkUp`. A state-specific collector starts at the parent of the scan root and walks upward one directory at a time:

1. check for `.stem`;
2. when absent, continue;
3. when readable and valid, add it to `Stems`;
4. when readable but malformed, add its parse failure to `ParseErrors` and continue;
5. stop after a successfully parsed stem with `root: true`;
6. otherwise stop at the filesystem root.

A malformed stem cannot reliably declare a root marker, so it cannot terminate discovery. Continuing upward preserves any valid governance context needed to classify the malformed layer.

The collector uses the normal filesystem because this amendment does not introduce the read-only filesystem abstraction tracked by #194. It remains pure discovery: no writes or temporary overlays occur.

## Data Flow

```text
report
  -> lexical candidate planning
  -> resolve opened AtomicTarget capabilities
  -> coalesce by physical target
  -> DiscoverStemState(report root)
       -> scan-owned stems and parse errors
       -> tolerant external ancestor collection
  -> Overlay(physical target, final candidate bytes)
  -> EvaluateStemState
       error diagnostic -> close all capabilities; no actions; no writes
       warn/info only   -> validated batch
  -> dry-run
       publish approved actions
       close capabilities
  -> real execution
       recheck parent identity
       stage, sync, chmod, and rename through opened parent
       publish actions only after success
       continue independent writes unless canceled
       close capabilities
```

Planning and real execution do not resolve the lexical target through separate code paths. Dry-run validates the same capability-bound identities but performs no mutation.

## Error Contract

### Target resolution failures

- An internal symlink whose physical parent remains within the allowed root is accepted.
- A symlink whose physical parent escapes the allowed root is rejected using the existing policy-rejection surface where applicable.
- A missing target file is allowed when its physical parent exists.
- A missing, unreadable, or unverifiable parent is rejected before prospective validation.
- Operational error messages name `reportTarget`, not an implementation-only physical path.

### Concurrent path changes

- Retargeting the lexical alias after capability creation does not redirect validation or execution.
- Moving, deleting, or replacing the physical parent makes the pre-write identity check fail without staging a file.
- Concurrent content replacement of the target or changes to other ancestor stems remain governed by the existing last-writer replacement semantics. This amendment does not add optimistic content locking or a multi-file transaction.

### Malformed external governance

A malformed governing ancestor produces a deterministic stem-health diagnostic such as:

```json
{
  "path": "../.stem",
  "check": "yaml-valid",
  "severity": "error",
  "message": "..."
}
```

The result is blocking in both report paths and both execution modes:

- non-zero exit;
- `complete: false`;
- `applied: []`;
- no writes;
- the diagnostic retained in `stem_health[]` and translated to deterministic `errors[]` under the existing version 1 contract.

Permission, cancellation, and other I/O failures remain operational errors rather than `yaml-valid` diagnostics.

### Resource cleanup

Every capability and staging file has a single owner. Cleanup runs on every return path. A failed target retains its original bytes and mode. After the prospective gate, independent later writes remain best-effort and are attempted unless the context is canceled.

## Testing Strategy

### `internal/fsx`

- Resolve an internal parent symlink to the expected physical target.
- Retarget the lexical alias after opening and prove the capability still addresses the original parent.
- Move or substitute the physical parent and prove the identity check blocks mutation.
- Reject a parent symlink escaping the allowed root.
- Preserve existing file mode.
- Preserve original bytes and remove staging files on injected failure.
- Close capabilities without leaking handles or making repeated cleanup unsafe.

### `internal/rules`

- Collect a valid external ancestor into `Stems`.
- Collect a malformed external ancestor into `ParseErrors`.
- Continue above malformed YAML to a valid `root: true` ancestor.
- Emit `yaml-valid` with a path relative to the report root.
- Preserve candidate evaluation ownership outside the scan root.
- Propagate cancellation and non-absence read errors.

### `cmd/rootline`

- Reproduce the internal-alias defect: a lexical child alias points to a physical `.stem` whose real parent hierarchy rejects the candidate.
- Retarget the alias between planning and execution and prove the write is not redirected.
- Coalesce two aliases resolving to the same physical `.stem` into one replacement.
- Assert proposal and analyze both block malformed `../.stem` governance with equivalent diagnostics.
- Assert dry-run/write parity and no premature actions.
- Preserve lexical report paths in operational errors.
- Preserve no-op, warning-only, duplicate-action, cancellation, and best-effort contracts.

### Release gates

- focused RED/GREEN regressions;
- affected packages with `-race`;
- `just check`;
- `just test`;
- `just coverage-check`;
- `go vet ./...`;
- governed documentation validation using the branch binary;
- the original parent-enum/child-string regression;
- `git diff --check`;
- final architecture review.

## Acceptance Criteria

1. Prospective validation and execution refer to the same physical target identity.
2. Retargeting a lexical alias cannot redirect an approved write.
3. Internal symlinks remain supported when their physical target is permitted.
4. Duplicate lexical aliases cause at most one physical replacement per target.
5. Every malformed governing ancestor is represented as structured `yaml-valid` stem health with a report-root-relative path.
6. Proposal and analyze produce equivalent blocking behavior.
7. Version 1 envelopes, dry-run parity, atomic-per-file writes, mode preservation, and best-effort multi-file execution remain unchanged.
8. The implementation does not introduce the general filesystem abstraction deferred to #194.

## Explicit Non-Goals

- No filesystem-wide injection interface.
- No writable virtual filesystem.
- No optimistic content locking.
- No multi-file rollback or journal.
- No envelope version bump.
- No change to monotonicity, merge, match, or document-validation semantics.
