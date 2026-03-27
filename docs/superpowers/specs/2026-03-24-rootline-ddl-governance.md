# Spec: Rootline as a DDL and Governance Layer

## Status

| Objective | Status | Implementation |
|-----------|--------|----------------|
| 1.1 Documentation | **Implemented** | README.md updated with DDL metaphor, database mapping table, domain types section |
| 1.2 Semantic Tagging | **Implemented** | `domain` property on `.stem` fields — see `2026-03-26-stem-domain-types-design.md` |
| 1.3 Governance Detectors | **Implemented** | 4 detectors in `analyze` pipeline: domain coverage, schema coverage, validation gaps, naming inconsistency |

## Context

Rootline is the **Data Definition Language (DDL)** for the filesystem-based database ecosystem. `.stem` files define schemas (tables → directories, columns → frontmatter fields, DDL → `.stem`). The `analyze` → `apply` pipeline runs 16 inference detectors (13 data + 3 governance) that suggest schema improvements and enforce DDL best practices.

Empirical evidence from 200+ sessions and the project's own documentation tree reveals concrete governance gaps:

| Gap | Evidence | Impact |
|-----|----------|--------|
| 23 directories without `.stem` files | E10, E13, F04, F14 in `docs/epics/` | Documents not governed — no validation, no constraints |
| Fields without `domain` declared | `estado`, `tipo` in existing `.stem` files have no semantic type | Consumer tools (Kedral, MCP) cannot resolve field meaning across projects |
| `tipo` inferred required but declared optional | `analyze` flags >80% usage | Schema understates actual constraint |
| Enum value drift | `estado` inferred `[Completed, Obsolete]` vs full schema values | Schema and usage diverge silently |

These are not data problems — they are schema governance problems. The engine computes correctly against whatever schema exists; the issue is that schemas themselves are incomplete, underspecified, or absent.

## Objective

Add 4 governance detectors to the `analyze` pipeline that flag DDL-level gaps — missing schemas, missing domains, insufficient validation rules, and structural inconsistencies. These detectors enforce that `.stem` schemas are **complete, semantically typed, and structurally consistent**.

Governance detectors follow the existing advisory model: `analyze` detects, `apply` corrects what it can, and flags the rest as `requires_agent` for human or LLM resolution.

## Design

### 1. Domain Coverage Detector

**File**: `internal/infer/domain_coverage.go`

**Purpose**: Flag `.stem` fields that lack a `domain` declaration. Every field with recognizable semantic meaning should have a domain — this is the rootline equivalent of SQL's `DOMAIN` or `CHECK` constraint annotations.

**Input**: Loaded stem rules (effective schema after merge).

**Logic**:
1. Walk all fields in the effective schema
2. For each field without `domain`: emit inference
3. Skip fields that are purely structural (type `section`) — sections don't carry semantic meaning in the domain sense

**Inference**:
```go
Inference{
    Type:    "missing_domain",
    Source:  "docs/epics/.stem",  // from SchemaField.Source — tracks which .stem defined the field
    Field:   "estado",
    Message: "Field 'estado' (type: enum, values: [draft, active, closed]) has no domain declared",
}
```

**Agent gating**: `requires_agent: true`. Assigning a domain requires understanding what the field *means*, not just its structure. The engine provides context (name, type, values); the agent or user decides the domain.

**Resolution path**: `rootline set <stem-path> --field <name>.domain <domain>` (already implemented).

**Incremental filter**: Covered if field already has `domain` set.

### 2. Schema Coverage Detector

**File**: `internal/infer/schema_coverage.go`

**Purpose**: Detect directories containing markdown files but governed by no `.stem` schema — not even via inheritance. These are "tables without DDL."

**Input**: Scan root path (filesystem walk).

**Logic**:
1. Walk directory tree from scan root
2. For each directory containing ≥1 `.md` file: attempt `.stem` walk-up discovery (same algorithm as `rules.LoadRulesForPath`)
3. If no `.stem` found anywhere in the walk-up chain to `.git` root → gap
4. If `.stem` found only via distant inheritance (≥3 levels up) → warn (schema is implicit, possibly unintentional)

**Inference types**:
```go
// No schema at all
Inference{
    Type:    "missing_schema",
    Source:  "docs/features/",
    Message: "Directory contains 5 markdown files but no .stem schema (checked walk-up to .git root)",
}

// Schema via distant inheritance (advisory)
Inference{
    Type:    "implicit_schema",
    Source:  "docs/epics/E10-intrinsic-hierarchy/F01-match-scoping/",
    Value:   "docs/epics/.stem",
    Message: "Directory inherits schema from docs/epics/.stem (3 levels up) — consider adding local .stem for explicit governance",
}
```

**Agent gating**: `missing_schema` is `requires_agent: false` — `apply` can generate a scaffold `.stem` by analyzing the union of frontmatter fields across the directory's markdown files (reuses existing `infer.Analyze()` logic). `implicit_schema` is `requires_agent: true` — deciding whether to add a local `.stem` requires judgment.

**Apply routing for `missing_schema`**: The existing `apply` command routes through `WalkUp(root)` which fails when no `.stem` exists — exactly the case `missing_schema` targets. To handle this:
- `cmd/rootline/apply.go` gains a **pre-phase** before the `WalkUp` call: scan the report for `missing_schema` inferences and process them first via a new `ScaffoldSchema(dirPath string, records []*extract.Record)` function.
- `ScaffoldSchema` creates a minimal version-2 `.stem` at the target directory with fields inferred from the union of frontmatter across the directory's markdown files. It reuses the `infer.Analyze()` field-type detection for type inference.
- The inference's `Source` field contains the directory path (e.g., `"docs/features/"`), which is the target for the new `.stem` file.
- After scaffolding, the remaining inferences proceed through the normal `WalkUp` → `ApplySchemaInferences` path (the newly created `.stem` is now discoverable).

**Incremental filter**: Covered if `.stem` exists in the directory (direct or ≤1 level up for `implicit_schema`).

### 3. Validation Gaps Detector

**File**: `internal/infer/validation_gaps.go`

**Purpose**: Detect fields declared in `.stem` that have insufficient validation rules — the equivalent of a column declared without `NOT NULL`, `CHECK`, or `FOREIGN KEY` constraints.

**Input**: Loaded stem rules (effective schema after merge) + extracted records (for usage analysis).

**Gaps detected**: Each gap uses a **distinct `Type`** value (not sub-types via `Value`) so the existing `agentRequiredTypes` map can gate them independently.

| Type | Condition | Severity | Agent? |
|------|-----------|----------|--------|
| `enum_without_values` | `type: enum` with no `values` list | error | true — agent must inspect data to propose values |
| `required_understatement` | Field used in >80% of records but not declared `required` | warn | true — needs judgment on whether it should be required |
| `untyped_field` | Field declared with no `type` and no `domain` (rootline cannot validate it) | error | false — can infer type from observed values |
| `sequence_incomplete` | `type: sequence` missing `prefix` or `digits` | error | false — can infer from existing ID patterns |

**Inference**:
```go
Inference{
    Type:    "enum_without_values",
    Source:  "docs/epics/.stem",
    Field:   "prioridad",
    Message: "Field 'prioridad' is declared as enum but has no values list — cannot validate",
}

Inference{
    Type:    "required_understatement",
    Source:  "docs/epics/.stem",
    Field:   "tipo",
    Message: "Field 'tipo' is used in 47/52 records (90%) but not declared required",
}
```

**Agent gating**: Per type (see table above). Each type is independently keyed in `agentRequiredTypes`.

**Deduplication with existing detectors**: The `enum_values` detector (existing) infers values from observed data. The `required_field` detector (existing) infers required from usage frequency. To avoid duplicate/conflicting inferences for the same field:
- Emit `enum_without_values` only when the field has `type: enum`, no `values` in the stem, AND the existing `enum_values` detector did not already produce an inference for that field. In practice: the validation gaps detector runs **after** the data-inference detectors and checks the accumulated report for coverage.
- Emit `required_understatement` only when the existing `required_field` detector did not already flag the field. Same mechanism — check prior categories in the report.

**Apply handlers**: `untyped_field` → add `type` based on observed values (reuses existing `applyFieldTypeNode` logic). `sequence_incomplete` → infer `prefix`/`digits` from existing IDs in the directory.

**Incremental filter**: Per type — `enum_without_values` covered if values exist; `required_understatement` covered if field is required; `untyped_field` covered if type or domain exists; `sequence_incomplete` covered if both prefix and digits are set.

**Relationship to stem-health**: The existing `enum-values` stem-health check validates that enums have ≥2 values. This detector catches the prior case: enums with **zero** values. The two are complementary — stem-health validates well-formed schemas; this detector catches schemas that aren't formed at all.

### 4. Structural Hygiene (extension)

**File**: `internal/infer/structural.go` (extend existing detector)

**Purpose**: Detect naming convention inconsistencies within directories — children that deviate from the dominant pattern.

**Input**: Scan root path (filesystem walk).

**Logic**:
1. For each directory with ≥3 children: analyze naming patterns
2. Identify dominant pattern (e.g., `E##-*` appears in 8/10 children)
3. If ≥70% of children match a pattern but outliers exist → flag outliers
4. Skip directories where no dominant pattern exists (< 70% consensus)

**Inference**:
```go
Inference{
    Type:    "naming_inconsistency",
    Source:  "docs/epics/",
    Value:   "E##-slug",
    Message: "8/10 children match 'E##-*' pattern; outliers: 'notes/', 'archive/'",
}
```

**Agent gating**: `requires_agent: true`. Deciding whether outliers should be renamed, excluded, or accepted requires judgment.

**Incremental filter**: Not applicable — structural checks always run fresh (filesystem state, not schema state).

### 5. Integration with analyze Pipeline

**Registration** in `cmd/rootline/analyze.go`:

Three new categories added to the `categories` slice:

```go
{"domain_coverage", "Domain Coverage", func() []infer.Inference {
    return infer.DetectMissingDomains(stemRules)
}},
{"schema_coverage", "Schema Coverage", func() []infer.Inference {
    return infer.DetectMissingSchemata(scanRoot)
}},
{"validation_gaps", "Validation Gaps", func() []infer.Inference {
    return infer.DetectValidationGaps(stemRules, records)
}},
```

Structural hygiene extends the existing `structural` category — no new registration needed.

**Agent-required types** added to `agentRequiredTypes` map:

```go
"missing_domain":            true,
"implicit_schema":           true,
"naming_inconsistency":      true,
"enum_without_values":       true,
"required_understatement":   true,
```

Note: `missing_schema`, `untyped_field`, and `sequence_incomplete` are NOT in this map — they have mechanical fixes and are applied automatically.

**Incremental filtering**: `FilterCoveredInferences` in `delta.go` extended with new `isCovered()` cases for each governance inference type (see per-detector sections above).

**Apply handlers**:
- `missing_schema` → pre-phase in `cmd/rootline/apply.go`: calls `ScaffoldSchema` to create new `.stem` file (see Section 2 for routing details)
- `untyped_field` → `internal/infer/apply.go`: reuses `applyFieldTypeNode` to add `type` to field in `.stem`
- `sequence_incomplete` → `internal/infer/apply.go`: adds `prefix`/`digits` to field in `.stem`

**Report JSON**: No schema changes. New categories appear as entries in `AnalyzeReport.Categories`. `version` remains 1.

### 6. Interaction with Existing Checks

| Existing mechanism | Governance detector | Relationship |
|-------------------|-------------------|--------------|
| stem-health `enum-values` | `enum_without_values` | Complementary — stem-health checks ≥2 values; governance catches 0 values |
| `enum_values` detector | `enum_without_values` | Deduplicated — governance only emits if data detector did not already infer values for the field |
| `required_field` detector | `required_understatement` | Deduplicated — governance only emits if data detector did not already flag the field |
| stem-health `domain-type-compat` | `missing_domain` | Sequential — first ensure domain exists, then validate compatibility |
| `structural.go` patterns | `naming_inconsistency` | Extension — existing pattern detection gains outlier reporting |
| `field_type` inference | `untyped_field` | Reuse — governance flags the gap, apply uses same `applyFieldTypeNode` logic |

## Files Modified

| File | Change |
|------|--------|
| `internal/infer/domain_coverage.go` (new) | Domain coverage detector |
| `internal/infer/schema_coverage.go` (new) | Schema coverage detector |
| `internal/infer/validation_gaps.go` (new) | Validation gaps detector |
| `internal/infer/structural.go` | Add naming inconsistency detection |
| `internal/infer/delta.go` | Incremental filter cases for new types |
| `internal/infer/apply.go` | Apply handlers for `missing_schema`, `untyped_field`, `sequence_incomplete` |
| `cmd/rootline/analyze.go` | Register 3 new categories, extend agent-required types |

## Verification

1. **Unit tests**: Each detector tested with synthetic `.stem` and filesystem fixtures
2. **Integration tests**: Full `analyze` pipeline includes governance categories; `--incremental` correctly filters covered inferences
3. **E2E tests**: Run `rootline analyze docs/epics/` and verify governance gaps appear for the 23 known directories without schemas and fields without domains
4. **Apply tests**: `rootline apply` correctly scaffolds `.stem` for `missing_schema`, adds types for `untyped_field`, skips agent-required inferences
5. **Backward compat**: Existing stems and documents pass all existing tests unchanged — governance detectors are additive
6. **Manual**: `rootline analyze --incremental docs/epics/` shows governance gaps that weren't visible before

## Out of Scope

- **Documentation updates** (README, CLAUDE.md, docs/) — deferred to separate session
- **Fuzzy search** — transversal capability, separate spec
- **Domain inference heuristics** — rejected as too fragile; domain assignment delegated to agent/user
- **Cross-reference repair** — 1,531 broken refs detected by existing analyzers; repair mechanism is a separate concern
