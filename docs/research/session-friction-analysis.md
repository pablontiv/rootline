---
estado: Pre-research
fecha: "2026-03-06"
metodo: session-analysis
---
# Session Friction Analysis

**Context**: Analysis of 200+ Claude Code sessions on the rootline project revealed recurring operational friction caused by feature gaps — not bugs, but missing capabilities that force manual workarounds. Some gaps are partially covered by existing research docs (disconnected pieces); others are entirely uncovered.

**Related**:
- [Intrinsic Hierarchy Principle](intrinsic-hierarchy-principle.md) — **depends-on**: Part 4 diagnoses the index file / estado drift problem (L278-323). Part 5 proposes v3 entity model as prerequisite (design blocked).
- [Inference Engine Architecture](inference-engine-architecture.md) — **borrows-pattern**: the `analyze` → report → `apply` → write-back pipeline is architecturally reusable for aggregate propagation.
- [Opportunity Areas](opportunity-areas.md) — **extends**: Item #7 (Watch Mode + Events) as a potential trigger mechanism for automatic propagation.

---

## Part 1 — Methodology

- **Source**: 200+ Claude Code session logs (`.jsonl` files) from the rootline project
- **Method**: Pattern extraction via keyword and categorical analysis of assistant messages
- **Period**: Full project history (Jan 2026 – Mar 2026)
- **Categories tracked**: `MANUAL_FRONTMATTER`, `AGGREGATE_ISSUE`, `FIX_INSUFFICIENT`, `PREPUSH_FAIL`, `OBSOLETE_AGGREGATE`
- **Scope**: Only friction caused by rootline limitations — not Claude Code issues, not coding bugs, not test failures

---

## Part 2 — Observed Friction Patterns

### P1: Manual Frontmatter Propagation

**Frequency**: 25+ sessions (most common pattern)

**What happens**: A subagent (or the main session) updates a child document's `estado` field (e.g., marking a Task as `Completed`). The parent README.md — which represents the directory as a record — retains its old `estado` value. Rootline detects the drift but cannot auto-fix it. The assistant must manually edit each parent README up the hierarchy.

**Workaround**: Open each parent README with `Edit`, change `estado` field manually. In deep hierarchies (Epic → Feature → Story), this means 3+ manual edits per completed task.

**Root cause**: Rootline can compute aggregate values (`derive`/`aggregate` in `.stem`) and detect drift (`DetectDrift()` in `drift.go`), but has no mechanism to **write computed values back** to file frontmatter. The compute→detect→fix pipeline is incomplete: compute and detect work, but fix only handles validation proposals (enum extension, field correction), not aggregate drift.

**Impact**: In the `/roadmap loop` workflow, where subagents complete multiple tasks per session, this creates a cascade of manual README updates that block the push workflow.

**Evidence** (session IDs): `5082262f`, `47dead07`, `1bccd4f9`, `ce0018c4`, `4c19f80a`, `7cf0b9f0`, `297fc3cd`, `4b323ce7`, `1082babc`, `46f49255`, `8c7a631d`, `34511435`, `b083c56d`, `8f49a24b`, `3efd8bd9`, `c0285677`, `14f520c8`, `6277ff74`, `327a1d9b`, `8384ef38`, `bcd77b1d`, `646fb74d`, `d96c4945`, `af365a50`, `75ab77ed`, `15df0162`

### P2: Rigid Aggregate Formulas

**Frequency**: 5 sessions

**What happens**: Aggregate expressions defined in `.stem` don't handle all terminal states. Most common case: the `estado` aggregate formula recognizes `Completed` as terminal but not `Obsolete`. When all children are `Obsolete`, the parent computes `Pending` — semantically incorrect.

**Workaround**: Either (a) manually set the parent `estado` to override the computed value, or (b) accept the incorrect computation.

**Root cause**: Aggregate formulas are project-defined expressions (`expr-lang/expr`). They work correctly for the states they enumerate, but new states (like `Obsolete`) require manual formula updates in `.stem`. The engine has no mechanism to suggest or auto-extend aggregate formulas when new enum values appear.

**Impact**: Creates confusion about project status — parents show "Pending" when all work is actually done (just marked obsolete instead of completed).

**Evidence** (session IDs): `5082262f`, `3ef33be8`, `7cf0b9f0`, `1082babc`, `327a1d9b`

### P3: Fix Scope Gap

**Frequency**: 4+ sessions (often co-occurs with P1)

**What happens**: `rootline fix` resolves validation errors (extend_enum, correct_value, add_field, migrate_value) but cannot resolve drift between computed aggregate values and stored frontmatter values. The user runs `rootline fix` expecting it to propagate aggregates; it does nothing because aggregate drift is outside its proposal scope.

**Workaround**: Manual editing after `rootline fix` reports no proposals.

**Root cause**: The proposal engine (`internal/proposal/`) generates proposals only from validation rule violations. Drift detection (`internal/rules/drift.go`) produces warnings, not fixable proposals. These are two separate systems with no bridge.

**Impact**: Creates a false expectation gap — users trust `rootline fix` to handle all fixable issues, but aggregate drift silently falls through.

**Evidence** (session IDs): `5082262f` (explicit: "`rootline fix` no corrigió el aggregate drift. Actualizo manualmente los 9 READMEs.")

### P4: Pre-push Validation Blocking

**Frequency**: 4 sessions

**What happens**: Git pre-push hooks run `rootline validate`. After subagent work (especially in worktrees), validation catches aggregate drift that `rootline fix` cannot resolve. Push is blocked. The session must manually fix all drift, re-commit, and retry push.

**Workaround**: Manual fix → `git add` → `git commit` → retry push.

**Root cause**: Cascading effect of P1 + P3. Validation correctly catches the drift, but the only resolution path is manual editing.

**Impact**: Blocks the push workflow at the worst possible moment — after all code work is done, tests pass, and the session is ready to finish.

**Evidence** (session IDs): `5082262f`, `3ef33be8`, `d7108412`, `4c19f80a`

---

## Part 3 — Capability Gap Analysis

### What rootline CAN do today

| Capability | Command | Status |
|------------|---------|--------|
| Compute per-record derived fields | `derive:` in `.stem` | Works |
| Compute bottom-up aggregates | `aggregate:` in `.stem` | Works |
| Detect drift between computed and stored values | `validate` (via `DetectDrift()`) | Works |
| Fix validation errors (enum, required, type) | `fix` | Works |
| Infer schema from existing data | `analyze` | Works |
| Write inferred schema to `.stem` | `apply` | Works |

### What rootline CANNOT do today

| Missing capability | Related pattern | Impact |
|--------------------|----------------|--------|
| Write computed aggregate values back to frontmatter | P1, P3 | 25+ sessions of manual editing |
| Auto-extend aggregate formulas for new enum values | P2 | Incorrect parent estado computation |
| Generate fixable proposals from aggregate drift | P3 | `fix` appears to do nothing |
| Coordinate multi-agent edits to shared hierarchy | P1, P4 | Worktree merges create drift |

### The Pipeline Gap

```
                    ┌──────────┐    ┌──────────┐    ┌──────────┐
  .stem defines:    │ derive:  │───>│ compute  │───>│ detect   │──> drift_warnings
  aggregate exprs   │aggregate:│    │ values   │    │ drift    │
                    └──────────┘    └──────────┘    └──────────┘
                                                         │
                                                         ▼
                                                   ┌──────────┐
                                         TODAY:    │  manual   │ <── P1: 25+ sessions
                                                   │  editing  │
                                                   └──────────┘
                                                         │
                                                         ▼
                                                   ┌──────────┐
                                         NEEDED:   │  write    │ <── feature gap
                                                   │  back     │
                                                   └──────────┘
```

---

## Part 4 — Coverage by Existing Research

### Intrinsic Hierarchy Principle (depends-on)

The diagnosis in Part 4 (L278-323) precisely describes the root cause:

> "The README is the Story. The T*.md files are its Tasks. The README's `estado` should be derived from the Tasks, not written by hand. But rootline has no way to know this because the schema doesn't declare the semantic relationship between the index file and its siblings." (L312)

Part 5 proposes the v3 entity model with per-entity `aggregate:` definitions as a prerequisite for scoped computation. However:
- The v3 design is **blocked** on a circular tension (declaring entity types re-declares hierarchy)
- Even if v3 unblocked, it defines **what** to compute, not **how to write it back**
- The write-back mechanism is a separate concern that works with v2 schemas too (if `aggregate:` is defined, the computed value can be written regardless of schema version)

**Conclusion**: The diagnosis is complete. The prerequisite is partially addressed (v2 `aggregate:` already works for computation). What's missing is the write-back step, which is independent of v3.

### Inference Engine Architecture (borrows-pattern)

The `analyze` → `AnalyzeReport` → `apply` pipeline demonstrates the write-back pattern:
1. `analyze` scans documents, produces a JSON report with proposals
2. `apply` reads the report and modifies `.stem` files and document frontmatter

This same pattern could serve aggregate propagation:
1. Compute aggregate values (already implemented in `derive` package)
2. Produce a propagation report (diff between computed and stored values)
3. Apply the report by rewriting frontmatter in index files

**Conclusion**: The pattern is reusable. The infrastructure exists (`internal/fix/` rewrites frontmatter, `internal/derive/` computes aggregates). What's missing is the orchestration that connects them.

### Opportunity Areas (extends)

Item #7 proposes Watch Mode + Events:

```bash
rootline watch docs/
# Emits events: document.created, document.updated, document.invalid, schema.changed
```

A `document.updated` event could trigger aggregate re-computation and write-back, making propagation automatic rather than manual. This would eliminate P1 entirely for workflows where rootline runs as a daemon.

**Conclusion**: Watch Mode is a trigger mechanism for the write-back feature — not a standalone solution. The write-back must work as an explicit command first; reactive mode can be layered on top.

### What's NOT covered by any existing doc

1. **Proposal generation from drift** — bridging `drift.go` warnings to `proposal/` fixable proposals
2. **Aggregate formula evolution** — auto-extending formulas when new enum values appear
3. **Multi-agent coordination** — how distributed editing (worktrees, subagents) interacts with hierarchical consistency

---

## Part 5 — Potential Features

### A: Aggregate Write-Back

A command (or flag) that computes aggregate values and writes them to index file frontmatter.

**Option A1**: `rootline fix --propagate` — extends existing `fix` to include aggregate drift in its scope.
- Pro: single command for all fixes. Users already expect `fix` to handle everything.
- Con: conflates two different operations (validation fix vs aggregate propagation).

**Option A2**: `rootline propagate [path]` — new standalone command.
- Pro: clear semantics. Can be composed with hooks/watch.
- Con: yet another command to learn.

**Option A3**: Extend `apply` — add aggregate propagation as a proposal type in `AnalyzeReport`.
- Pro: reuses existing infrastructure.
- Con: `analyze`/`apply` is for schema inference, not runtime state management. Semantic overloading.

**Open questions**:
- Should propagation be recursive (full hierarchy) or single-level?
- Dry-run mode? (show what would change before writing)
- How to handle fields where stored value was intentionally set (manual override vs computed)?

### B: Expanded Fix Scope

Bridge drift detection to the proposal engine, so `rootline fix` generates proposals for aggregate drift.

**Mechanism**: When `DetectDrift()` finds a mismatch between computed aggregate and stored value, generate a `correct_value` proposal with source `"aggregate_drift"`.

**Open questions**:
- Should drift proposals have lower confidence than validation proposals?
- How to distinguish "stored value is wrong" from "stored value is an intentional override"?

### C: Aggregate Formula Extensibility

When new enum values appear (e.g., `Obsolete` added to `estado`), the engine could:
1. Detect that the aggregate formula doesn't handle the new value
2. Suggest formula updates
3. Or: treat unknown values as a specific fallback (configurable)

**Open questions**:
- Should formulas be self-extending, or should `analyze` detect the gap?
- Default behavior for unhandled enum values: error, warning, or passthrough?

### D: Multi-Agent Workflow Support

When multiple agents edit documents in the same hierarchy (especially via worktrees), consistency is compromised because each agent only sees its local state.

**Potential mechanisms**:
- Post-merge hook that runs `rootline propagate` automatically
- Lock mechanism for hierarchical scopes
- Event-driven re-computation (ties into Watch Mode from opportunity-areas #7)

**Open questions**:
- Is this rootline's responsibility, or the orchestrator's (Claude Code skills)?
- What's the minimal viable approach? (likely: post-merge hook + propagate command)

---

## Part 6 — Open Questions

1. Should aggregate write-back be a new command, a `fix` extension, or an `apply` extension?
2. Should propagation be recursive (full hierarchy) or single-level?
3. How to handle intentional overrides vs stale values?
4. Should this be automatic (watch mode / hooks) or explicit (manual command)?
5. What's the interaction with `derive:` (per-record) vs `aggregate:` (cross-record)?
6. Does the worktree/subagent pattern need rootline-level support, or is it an orchestration concern?
7. Can the write-back feature work with v2 schemas, or does it require v3 entity model?
8. Should drift proposals have different confidence levels than validation proposals?
