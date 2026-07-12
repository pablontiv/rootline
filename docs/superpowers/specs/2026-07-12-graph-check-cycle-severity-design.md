# Design: Opt-in cycle failure in `rootline graph --check`

**Date**: 2026-07-12
**Status**: Approved
**Breaking**: Yes (`feat!`)

## Problem

`graph --check` cannot gate documentation wikis because cycle detection is
hardcoded as a failure:

```go
// cmd/rootline/graph.go:102
hasProblems := len(cycles) > 0 || len(broken) > 0
```

On the DS Prima wiki (`/Users/Shared/dsprima/docs`, `.stem` declares
`links.styles: [markdown]`, 72 nodes / 200 edges), the 22 remaining cycles are
all legitimate cross-references (master doc ↔ diagrams ↔ domain indexes,
sibling docs referencing each other). For narrative documentation, link cycles
are the norm, not a defect — cycles-as-error only makes sense for
dependency/schema graphs. Result: `--check` exits 1 permanently even with zero
broken links, so it can never gate anything.

## Decision

Add `cycles` as a fourth key in `links.checks`, symmetric with the existing
opt-in keys (`resolve`, `anchors`, `encoding`): absent or `false` means cycles
are reported as informational only; `true` means cycles fail `--check`.

```yaml
# .stem
links:
  styles: [markdown]
  checks:
    resolve: true
    anchors: true
    encoding: true
    cycles: true    # opt-in: cycles fail --check. Absent = informational.
```

### Why this shape

Two alternatives were considered and rejected:

- **Asymmetric bool** (`cycles: false` demotes, absent = fail): preserves
  backcompat but inverts the default semantics of the `checks` block, where
  every other key is opt-in. Rejected for the asymmetry.
- **Separate severity key** (`links.cycles: error|info`): self-documenting and
  symmetric, but adds a second configuration surface for the same concern.
  Rejected in favor of keeping everything under `checks`.

### Breaking change, accepted deliberately

Today cycles always fail `--check`. After this change, a repo without
`checks.cycles: true` no longer fails on cycles. This is technically breaking,
but nobody can be using `graph --check` as a gate today — the permanent exit 1
on cyclic repos is precisely the bug motivating this change. A repo that wants
cycle gating adds one line. Pre-1.0 semver: `feat!` bumps minor
(v0.x → v0.(x+1)).

## Changes

### 1. Schema — `internal/rules/rules.go`

`LinkChecks` gains a plain bool field:

```go
type LinkChecks struct {
	Resolve  bool `yaml:"resolve" json:"resolve,omitempty"`
	Anchors  bool `yaml:"anchors" json:"anchors,omitempty"`
	Encoding bool `yaml:"encoding" json:"encoding,omitempty"`
	Cycles   bool `yaml:"cycles" json:"cycles,omitempty"`
}
```

No pointer, no custom parsing — yaml decoding handles it. No changes to merge
semantics or `IsEmpty()` (the `Checks` struct already participates in both).

### 2. CLI — `cmd/rootline/graph.go`

`runGraph` already loads the merged stem (`rules.WalkUp` +
`rules.MergeStemFiles`). Derive the effective setting from it:

```go
failCycles := stem != nil && stem.Links.Checks != nil && stem.Links.Checks.Cycles
```

New flag `--fail-cycles` (bool). When set explicitly (detected via
`cmd.Flags().Changed("fail-cycles")`), it overrides the `.stem` value in either
direction.

Exit logic becomes:

```go
hasProblems := (failCycles && len(cycles) > 0) || len(broken) > 0
```

Output:

- Cycles in failing mode: current header `Cycles found: N`.
- Cycles in informational mode: header `Cycles found (informational): N`,
  same per-cycle lines.
- Broken links: unchanged.
- No cycles and no broken links: current message unchanged.
- JSON contract unchanged: `cycles` array always present and fully populated
  regardless of severity.

### 3. Tests

Unit (`internal/rules`):

- `checks.cycles: true` decodes to `Cycles == true`.
- `checks.cycles: false` and absent key decode to `Cycles == false`.

E2E (`internal/e2e`):

- (a) Cyclic repo, no `cycles` key → exit 0, cycles printed as informational.
- (b) Cyclic repo, `cycles: true` → exit 1.
- (c) No `cycles` key, broken link present → exit 1 (informational cycles do
  not mask broken links).
- (d) `--fail-cycles` overrides the stem in both directions:
  `--fail-cycles=true` with no key fails on cycles; `--fail-cycles=false` with
  `cycles: true` does not.

## Acceptance criteria

1. On `/Users/Shared/dsprima/docs` (no `cycles` key): `rootline graph --check`
   exits 0 (zero broken links), still prints the 22 cycles as informational,
   JSON `cycles` array unchanged.
2. Same repo plus untracked canary `[x](no-existe.md)`: exits 1, broken link
   reported.
3. Repo wanting cycle gating sets `checks.cycles: true` and gets today's
   failing behavior.
4. E2E tests cover the cases above.

## Out of scope

- Warning on unknown keys inside `links.checks` (silently-inert config keys).
  Separate small fix in the same area.

## Commit

`feat!: make graph --check cycle failure opt-in via checks.cycles` with a
`BREAKING CHANGE:` footer describing the default flip.
