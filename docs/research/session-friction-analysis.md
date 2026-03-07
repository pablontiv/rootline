---
estado: Pre-research
fecha: "2026-03-06"
metodo: session-analysis
---
# Session Friction Analysis

**Context**: Analysis of 200+ Claude Code sessions on the rootline project revealed recurring operational friction caused by feature gaps — not bugs, but missing capabilities that force manual workarounds. This document is the primary problem definition, grounded in empirical evidence. The documents below contain partial prior work that overlaps with the friction patterns identified here, but none covers the full problem space or proposes an integrated solution.

**Related**:
- [Intrinsic Hierarchy Principle](intrinsic-hierarchy-principle.md) — **prior-work**: Part 4 (L278-323) diagnosed the same drift problem from a principle-based angle. Part 5 proposes v3 entity model as one solution path (design blocked on circular tension).
- [Inference Engine Architecture](inference-engine-architecture.md) — **reusable-pattern**: the `analyze` → report → `apply` → write-back pipeline is architecturally reusable for aggregate propagation.
- [Opportunity Areas](opportunity-areas.md) — **undeveloped-idea**: Item #7 (Watch Mode + Events) sketches a trigger mechanism that this document develops further as a reactive propagation layer.

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

**Root cause**: `AggregateAll()` computes the correct value in `record.Derived` (in-memory only). `DetectDrift()` catches the mismatch. But there is no bridge to `Proposal` → `ApplyProposals()` → `RewriteFrontmatter()`, which is the existing write-back infrastructure. See Part 3 for technical details.

**Observed in session** (`5082262f`):
> "`rootline fix` no corrigió el aggregate drift. Actualizo manualmente los 9 READMEs."

**Evidence** (26 session IDs): `5082262f`, `47dead07`, `1bccd4f9`, `ce0018c4`, `4c19f80a`, `7cf0b9f0`, `297fc3cd`, `4b323ce7`, `1082babc`, `46f49255`, `8c7a631d`, `34511435`, `b083c56d`, `8f49a24b`, `3efd8bd9`, `c0285677`, `14f520c8`, `6277ff74`, `327a1d9b`, `8384ef38`, `bcd77b1d`, `646fb74d`, `d96c4945`, `af365a50`, `75ab77ed`, `15df0162`

### P2: Rigid Aggregate Formulas

**Frequency**: 5 sessions

**What happens**: The `estado` aggregate expression in `docs/epics/.stem` enumerates specific states but doesn't handle `Obsolete`:

```
all(descendants, {.estado == "Completed"}) ? "Completed" :
any(descendants, {.estado == "Blocked"}) ? "Blocked" :
any(descendants, {.estado == "On Hold"}) ? "On Hold" :
any(descendants, {.estado == "In Progress"}) ? "In Progress" :
any(descendants, {.estado == "Specified"}) ? "Specified" :
"Pending"
```

When all children are `Obsolete`, none of the conditions match → falls through to `"Pending"`. Semantically incorrect.

**Observed in session** (`3ef33be8`):
> "Ahí está el problema: la fórmula de agregación solo reconoce 'Completed' como estado terminal, no 'Obsolete'. Todos los descendants son Obsolete, ninguno cae en los cases definidos, y el default es 'Pending'."

**Workaround**: Either (a) remove `estado` from parent READMEs entirely, or (b) manually override.

**Root cause**: Aggregate formulas are project-defined `expr-lang/expr` expressions. The engine has no mechanism to detect that a formula doesn't cover all enum values or to suggest formula updates when new values appear.

**Evidence** (5 session IDs): `5082262f`, `3ef33be8`, `7cf0b9f0`, `1082babc`, `327a1d9b`

### P3: Fix Scope Gap

**Frequency**: 4+ sessions (often co-occurs with P1)

**What happens**: `rootline fix` resolves validation errors (12 proposal types: `extend_enum`, `correct_value`, `add_field`, `migrate_value`, etc.) but cannot resolve drift between computed aggregate values and stored frontmatter values. Users run `rootline fix` expecting it to handle aggregate drift; it does nothing.

**Observed in sessions**:
> "Fix doesn't handle drift. Let me manually update the READMEs." (`ce0018c4`)
> "The fix tool doesn't handle aggregates. Let me manually update the estado." (`7cf0b9f0`)
> "The fix tool has an issue with a missing local `.stem`. Let me just manually fix the aggregate mismatches." (`297fc3cd`)

**Root cause**: The proposal engine (`internal/proposal/`) generates proposals from validation rule violations. The existing `infer_from_children` proposal type (proposal.go L403-471) only fires when `estado` is **missing** (a `required` rule violation) — not when it **exists but has a stale value**. Drift detection produces `DriftWarning` objects that are separate from the `Proposal` pipeline.

**Evidence** (session IDs): `5082262f`, `ce0018c4`, `7cf0b9f0`, `297fc3cd`

### P4: Pre-push Validation Blocking

**Frequency**: 4 sessions

**What happens**: Git pre-push hooks run `rootline validate`. After subagent work (especially in worktrees), validation catches aggregate drift that `rootline fix` cannot resolve. Push is blocked after all code work is done and tests pass.

**Observed in session** (`5082262f`):
> "Pre-push hook falla con validación de rootline. Veamos qué errores hay."
> (Followed by:) "Los agentes marcaron los tasks como Completed pero no actualizaron los READMEs padre (aggregated estado)."

**Root cause**: Cascading effect of P1 + P3.

**Evidence** (session IDs): `5082262f`, `3ef33be8`, `d7108412`, `4c19f80a`

---

## Part 3 — Technical Pipeline Analysis

### The Existing Pipeline

Five components participate in the compute→detect→fix cycle. Four work. The fifth is missing.

#### 1. Aggregate Computation (`internal/derive/aggregate.go`)

`AggregateAll()` walks index files bottom-up, evaluates `aggregate:` expressions from `.stem`, and stores results in `record.Derived`:

```go
// aggregate.go L107-157
func aggregateRecord(record, effective, descendants, children) {
    // ... evaluates expr-lang expression ...
    result, err := ev.Eval(compiled, env)
    record.Derived[field] = result   // ← in-memory only, never persisted
}
```

The computation is correct. The value lives in `record.Derived["estado"]` but is never written to disk.

#### 2. Drift Detection (`internal/rules/drift.go`)

`DetectDrift()` compares parent frontmatter against child values. It does NOT use the aggregate formula — it checks raw field unanimity:

```go
// drift.go L27-74
func DetectDrift(parent, children, schema) []DriftWarning {
    // For each shared field (no match restriction):
    //   if ALL children unanimously agree on a value
    //   AND that value differs from parent's stored value
    //   → DriftWarning
}
```

Limitations:
- Only detects drift when children are **unanimous**. Mixed states (3 Completed + 1 In Progress) produce no warning.
- Returns `DriftWarning`, not `Proposal` — a separate type with no bridge to the fix pipeline.

#### 3. Proposal Engine (`internal/proposal/`)

Generates fixable `Proposal` objects from validation errors. 12 types defined:

```go
// proposal.go L20-34
ExtendEnum, MigrateValue, CorrectValue, ExtractBody,
InferFromChildren, AddField, CorrectLink, InferFromSiblings,
CorrectOutlier, AddAggregate, RemoveStemField
```

`InferFromChildren` (L403-471) is the closest to aggregate propagation, but only handles **missing** fields:

```go
// Only fires when estado is absent (required rule violation)
for _, e := range pathErrs {
    if e.Rule != "required" || e.Field != "estado" {
        continue  // ← skips stale values entirely
    }
    inferred := InferEstado(childEstados)
    // → generates Proposal
}
```

A parent README with `estado: Specified` (stale but present) will never trigger this code path.

#### 4. Fix Application (`internal/fix/fix.go`)

`ApplyProposals()` takes `Proposal` objects and rewrites frontmatter on disk:

```go
// fix.go L21-25
func ApplyProposals(ctx, report, root, records) error {
    // 1. Apply extend_enum proposals (modify .stem)
    // 2. Apply data-level proposals (modify frontmatter via RewriteFrontmatter)
}
```

`RewriteFrontmatter()` (L158+) rebuilds the markdown file with updated YAML frontmatter. This infrastructure works and is well-tested.

#### 5. The Missing Bridge

```
AggregateAll()                           ApplyProposals()
     │                                        │
     ▼                                        ▼
record.Derived["estado"] = "Completed"   RewriteFrontmatter(fm)
                                              │
         ❌ NO BRIDGE EXISTS ❌                ▼
                                         file written to disk
DetectDrift()
     │
     ▼
DriftWarning{ParentValue: "Specified", ChildrenValue: "Completed"}
     │
     ❌ DriftWarning ≠ Proposal (different types, different pipelines)
```

The computed value is correct (left side). The write mechanism exists (right side). What's missing is the conversion: `Derived[field] ≠ Frontmatter[field]` → `Proposal{Type: CorrectValue, From: stored, To: computed}`.

### The Aggregate Formula Gap

The `docs/epics/.stem` aggregate expression (L60-66) explicitly enumerates 5 states: Completed, Blocked, On Hold, In Progress, Specified. Any value outside this set — including `Obsolete` — falls through to the default `"Pending"`.

The engine cannot detect that the formula is incomplete relative to the `estado` enum. If `estado` has 7 enum values but the aggregate only handles 5, there is no warning. When a new value appears (user adds `Obsolete`), the formula silently produces incorrect results.

---

## Part 4 — Coverage by Existing Research

### Intrinsic Hierarchy Principle (prior-work)

Part 4 (L278-323) diagnosed the identical problem:

> "The README is the Story. The T*.md files are its Tasks. The README's `estado` should be derived from the Tasks, not written by hand. But rootline has no way to know this because the schema doesn't declare the semantic relationship between the index file and its siblings." (L312)

Part 5 proposes a v3 entity model with per-entity `aggregate:` definitions. However:
- The v3 design is **blocked** on a circular tension ("don't redeclare hierarchy" vs "engine needs entity types")
- Even if unblocked, it defines **what** to compute, not **how to write it back**
- The write-back mechanism works with v2 schemas: if `aggregate:` is defined, the bridge can operate regardless of schema version

**Conclusion**: The diagnosis is complete. The prerequisite (aggregate computation) already works in v2. What's missing — the bridge — is independent of v3.

### Inference Engine Architecture (reusable-pattern)

The `analyze` → `AnalyzeReport` → `apply` pipeline demonstrates the write-back pattern:
1. `analyze` scans documents, produces a JSON report with typed proposals
2. `apply` reads the report and modifies `.stem` files and document frontmatter

The same proposal-based architecture can serve aggregate propagation. The `CorrectValue` proposal type already exists and writes to frontmatter.

**Conclusion**: The infrastructure is reusable. No new write mechanism needed — only a new proposal source.

### Opportunity Areas (undeveloped-idea)

Item #7 proposes Watch Mode + Events:

```bash
rootline watch docs/
# Emits events: document.created, document.updated, document.invalid, schema.changed
```

A `document.updated` event could trigger aggregate re-computation + write-back automatically. This would layer reactive propagation on top of the explicit command.

**Conclusion**: Watch Mode is a trigger layer, not a solution. The bridge must work as an explicit command first; reactive mode is additive.

### Gaps not covered by any existing doc

1. **Aggregate formula completeness** — no doc proposes detecting when a formula doesn't cover all enum values
2. **Stale value correction** — `infer_from_children` only handles missing values; no mechanism for stale values
3. **Multi-agent coordination** — no doc addresses how worktree/subagent workflows interact with hierarchical consistency

---

## Part 5 — Proposed Solutions

### Solution A: Bridge DriftWarning → Proposal (minimum viable)

Add a function that compares `record.Derived` against `record.Frontmatter` for index files and generates `CorrectValue` proposals:

```go
// New function in internal/proposal/ or internal/fix/
func PropagateAggregates(records []*extract.Record, root string) []Proposal {
    // 1. Run AggregateAll() — populates record.Derived for all index files
    derive.AggregateAllSimple(ctx, records, root)

    var proposals []Proposal
    for _, rec := range records {
        if !isIndexFile(rec) { continue }
        for field, computed := range rec.Derived {
            stored, has := rec.Frontmatter[field]
            if !has || fmt.Sprint(stored) == fmt.Sprint(computed) { continue }
            proposals = append(proposals, Proposal{
                Type:  CorrectValue,
                Field: field,
                From:  fmt.Sprint(stored),
                To:    fmt.Sprint(computed),
                Paths: []string{rec.Path},
                Description: fmt.Sprintf("propagate %s: %v → %v (from aggregate)", field, stored, computed),
            })
        }
    }
    return proposals
}
```

This reuses:
- `derive.AggregateAllSimple()` — already computes correct values
- `proposal.CorrectValue` type — already defined
- `fix.ApplyProposals()` → `fix.RewriteFrontmatter()` — already writes to disk

**Estimated scope**: ~50 lines of Go + CLI flag wiring.

### Solution B: `rootline fix --propagate`

Extend `cmd/rootline/fix.go` to include aggregate propagation when `--propagate` flag is set:

1. Run normal proposal analysis (existing behavior)
2. If `--propagate`: also run `PropagateAggregates()` from Solution A
3. Merge proposals and apply together

This keeps `fix` as the single "make it right" command while adding opt-in aggregate propagation.

### Solution C: Aggregate Formula Completeness Check

Add a stem-health diagnostic that compares aggregate expression coverage against enum values:

1. Parse the `aggregate:` expression for the field
2. Extract string literals from the expression (e.g., `"Completed"`, `"Blocked"`)
3. Compare against the field's `enum.values` list
4. If any enum value is not referenced in the expression → warning

This would catch the `Obsolete` gap at schema validation time rather than at runtime.

### Solution D: Post-merge Hook (multi-agent coordination)

A git post-merge hook that automatically runs `rootline fix --propagate`:

```bash
#!/bin/sh
# .githooks/post-merge
rootline fix --propagate docs/epics/ 2>/dev/null || true
```

This eliminates P4 (pre-push blocking after worktree merges) by propagating aggregates before the developer even attempts to push.

---

## Part 6 — Open Questions

1. Should `--propagate` be a flag on `fix` or a standalone command (`rootline propagate`)?
2. Should propagation run by default in `fix`, or always require opt-in?
3. How to handle intentional manual overrides? (e.g., user deliberately sets `estado: On Hold` on a parent even though children are all `Completed`)
4. Should aggregate formula completeness be a stem-health check (warning) or a validation error?
5. Can `DetectDrift` be enhanced to compare against `Derived` values (from aggregate computation) instead of requiring unanimous children?
6. Should the bridge also handle `derive:` fields (per-record expressions), or only `aggregate:` fields?
