---
estado: Implemented
fecha: "2026-03-02"
metodo: collaborative-research
---
# Inference Engine Architecture

**Context**: Analysis of rootline's `init` command against a real-world dataset (homeserver/automation/docs/epics — 375 files, 4 hierarchy levels: E##→F##→S###→T###) revealed significant gaps between what `init` infers and what a complete `.stem` requires. This research proposes an architecture for distributing inference responsibilities across three layers: the Go engine, Claude Code skills, and LLM agents.

**Related**: [Intrinsic Hierarchy Principle](intrinsic-hierarchy-principle.md) — explores the v2→v3 schema model. This document focuses on inference, not schema format.

---

## Part 1 — Current State of Inference

### What `init` Does Today

`rootline init` scans existing markdown files, analyzes their frontmatter, and generates a `.stem` schema. It operates in two modes:

| Mode | Trigger | Output |
|------|---------|--------|
| Flat | <2 hierarchy levels detected | `version: 2` `.stem` with flat schema |
| Hierarchical | ≥2 levels with pattern `[A-Z]+\d+-` | `version: 2` `.stem` with `match:` conditions |

**Source files**:
- `cmd/rootline/init.go` — CLI command, orchestrates modes
- `internal/infer/infer.go` — `Analyze()`: field detection, type inference, enum/required heuristics
- `internal/infer/hierarchy.go` — `AnalyzeHierarchy()`: level detection, field distribution, sequence inference
- `internal/migrate/aggregate.go` — `GenerateAggregates()`: auto-generated aggregate expressions for enum fields

### Inference Heuristics

| What | How | Threshold |
|------|-----|-----------|
| Field type: enum | ≤20 unique values AND present in >50% of records | Hardcoded in `Analyze()` |
| Field type: list | Value is an array | Type check |
| Field type: string | Default fallback | N/A |
| Required | Present in >80% of records | Hardcoded in `Analyze()` |
| Sequence | Directory names match `^([A-Z]+)(\d+)-` with ≥2 occurrences at same depth | Regex + count |
| Aggregate | Enum fields in hierarchical mode; values classified by semantic keywords (completado→terminal, bloqueada→negative) | Keyword matching in `classifyEnumValues()` |

### Hierarchy Detection Is Generic

The pattern detector (`internal/infer/hierarchy.go:15`) uses:

```go
var dirPattern = regexp.MustCompile(`^([A-Z]+)(\d+)-`)
```

This matches ANY uppercase prefix + digits + dash. The E/F/S/T pattern is not hardcoded — it emerges from analyzing actual directory names. Detection requires ≥2 directories with the same (prefix, digits) at the same depth, and ≥2 levels total.

Path handling uses `filepath.ToSlash()` for cross-platform compatibility (Windows, Linux, macOS).

---

## Part 2 — Thirteen Inference Categories

Analysis of the homeserver dataset identified 13 categories of information a complete `.stem` needs. Only the first 3 are currently implemented.

### Categories Operating on Frontmatter

| # | Category | Signal | Status |
|---|----------|--------|--------|
| 1 | Schema (types, enums, required) | Value frequency, cardinality | **Implemented** |
| 2 | Sequences (prefix, digits) | Directory naming patterns | **Implemented** |
| 3 | Aggregates (enum rollup) | Semantic keyword classification | **Implemented** |
| 4 | Structural (require_index, min_children) | Filesystem stat: README.md presence, child count | Not implemented |
| 5 | Links (allowed types, target patterns) | Wiki-link extraction from body (extractor exists) | Not implemented |

### Categories Operating on Body Content

| # | Category | Signal | Status |
|---|----------|--------|--------|
| 6 | Body structure (required sections) | Recurring headings per level (e.g., `## Contexto` in >90% of tasks) | Not implemented |
| 7 | Back-references | Links to `../README.md` as parent reference | Not implemented |
| 8 | Constants | Field with single value across 100% of records at a level | Not implemented (data exists but not flagged) |
| 9 | Heterogeneous dependencies | `[[blocks:]]` vs `## Dependencias` text vs `## Orden de Ejecucion` tables | Not implemented |
| 10 | Cross-epic references | Path notation `E##/F##/S###/T###` in body text | Not implemented |
| 11 | Traceability (Contribuye a) | Body text referencing parent's acceptance criteria | Not implemented |
| 12 | Invariants (Preserva) | `## Preserva` sections with `INV#:` pattern + verification commands | Not implemented |
| 13 | Sub-schema by type | YAML code blocks in `## Especificacion Tecnica` correlated with `tipo` field | Not implemented |

---

## Part 3 — Three-Layer Responsibility Model

Each inference category belongs to the layer best suited for its nature.

### Layer 1: Rootline Engine (Go)

**Nature**: Deterministic computation. No ambiguity, reproducible results.

| # | Category | Why engine |
|---|----------|-----------|
| 1 | Schema | Counting + thresholds |
| 2 | Sequences | Regex on directory names |
| 3 | Aggregates | Keyword classification |
| 4 | Structural | Filesystem stat (README.md exists? child count?) |
| 7 | Back-references | Pattern match: link target is `../README.md` |
| 8 | Constants | Single value in 100% of records — trivial to detect |
| 5' | Co-occurrence | When `tipo=X`, field Y present in N% of cases — statistical but deterministic |

These are operations with well-defined inputs and outputs. No reasoning required.

### Layer 2: Skills (Codified Procedures)

**Nature**: Fixed procedure that combines multiple rootline calls + domain logic with thresholds.

| # | Category | Why skill |
|---|----------|----------|
| 5 | Links | Run `rootline graph` to extract wiki-links, group targets by pattern (`T###` vs `F##`), generate `links:` section. Procedure is fixed; deciding `allowed: [blocks, reference]` needs domain context. |
| 6 | Body structure | Read N files per level, extract headings with regex, compute frequency, propose required sections. Procedure is fixed; the threshold for "required" needs judgment. |
| 10 | Cross-epic refs | Extract `E##/F##/...` patterns from body, normalize, propose formalization as wiki-links. |
| 5' | Validate requires | Consume co-occurrence data from engine, generate candidate `requires` rules. |

Skills are the escape from the "prompt engineering hamster wheel" (ref: Towards Data Science, "Claude Skills and Subagents"). The procedure is codified once and executed N times reproducibly, instead of an LLM re-discovering the inference logic each session.

### Layer 3: Agents (Semantic Reasoning)

**Nature**: Ambiguity that cannot be resolved by fixed rules. Requires understanding context.

| # | Category | Why agent |
|---|----------|----------|
| 9 | Heterogeneous deps | Distinguishing "T001-T004 (all running)" as dependency vs casual mention requires contextual comprehension. |
| 11 | Traceability | Matching "Contribuye a: Restore exitoso + notificacion ntfy" against parent acceptance criteria requires semantic matching, not string matching. |
| 12 | Invariants | Extracting INV# is regex, but evaluating redundancy between invariants requires reasoning. |
| 13 | Sub-schema by type | Inferring that `tipo=k8s-workload` implies {namespace, image, resources} in embedded YAML requires induction from N examples — program synthesis. |

Agent architecture (single generalista vs multiple specialized) is an open question. The concrete boundary is: categories that can be codified as procedure (skills) vs categories that require reasoning (agents).

### Compositional Delegation

The three layers compose:

```
Agent (reasons about ambiguity)
  └── Skill /infer-links (fixed procedure)
  │     └── rootline graph (extracts wiki-links)
  │     └── rootline query (groups by pattern)
  │     └── generates links: section
  └── Skill /infer-validates (fixed procedure)
  │     └── rootline analyze (co-occurrence data)
  │     └── generates validate: rules candidates
  └── Agent decides: is "Dependencias" section a formal link or free text?
```

---

## Part 4 — The Descriptive vs Normative Barrier

### The Fundamental Limit

Categories 1-8 are **observational** — they infer rules that the data already complies with. But the most valuable rules are the ones the data **should** comply with and doesn't.

Example from the homeserver dataset: 11 parent READMEs had `estado: In Progress` while all their children were `Completado`. This is a **normative** violation — the data doesn't comply with a rule that should exist. But `init` cannot infer the rule because the data itself violates it.

`init` can discover the **descriptive** schema (what exists). It cannot discover the **normative** schema (what should exist). The gap between the two is where the human enters — or where a v3 schema makes certain normative properties automatic by default (like vertical consistency, per the Intrinsic Hierarchy Principle research).

### Implications for the Report

The inference report should clearly separate:
- **Descriptive findings**: "field X appears in 86% of records" — observable fact
- **Normative suggestions**: "field X should be required" — interpretation that needs human/agent confirmation

Rootline engine produces descriptive findings. Skills and agents produce normative suggestions. The report carries both, labeled distinctly.

---

## Part 5 — The Inference Report (`analyze`)

### Design Principles

1. **Porcentual evidence, not confidence scores** — rootline reports `presence: 0.86`, not `confidence: high`. The consumer (agent, skill, human) decides if 86% is sufficient for `required: true`.
2. **Dual audience** — JSON for agents/skills, human-readable for `--output table`.
3. **Actions, not opinions** — each inference proposes a concrete action (`add_field`, `add_validate_rule`, `extend_enum`, `review`) with evidence.

### Report Format

```json
{
  "version": 1,
  "scan": {
    "root": "/path/to/docs/epics",
    "total_files": 375,
    "total_directories": 131,
    "hierarchy_detected": true,
    "levels": ["E(2)", "F(2)", "S(3)", "T(3)"]
  },
  "inferences": [
    {
      "category": "schema",
      "action": "add_field",
      "field": "estado",
      "type": "enum",
      "values": ["Pending", "Completado", "Bloqueada"],
      "evidence": {
        "presence": 1.0,
        "unique_values": 3,
        "files_with": 375,
        "files_total": 375
      },
      "target": "schema"
    },
    {
      "category": "structural",
      "action": "add_structural_rule",
      "rule": "require_index",
      "value": "README.md",
      "evidence": {
        "directories_with_index": 58,
        "directories_with_children": 58,
        "presence": 1.0
      },
      "target": "schema"
    },
    {
      "category": "co_occurrence",
      "action": "add_validate_rule",
      "rule": "requires",
      "if": {"tipo": "ci-cd"},
      "then": {"fields": ["ejecutable_en"]},
      "evidence": {
        "co_occurrence": 0.86,
        "matches": 12,
        "total": 14
      },
      "target": "schema"
    },
    {
      "category": "constant",
      "action": "review",
      "field": "ejecutable_en",
      "value": "1 sesion",
      "evidence": {
        "presence": 1.0,
        "unique_values": 1,
        "level": "T*"
      },
      "target": "info"
    },
    {
      "category": "drift",
      "action": "correct_value",
      "file": "E04/F10/S006/README.md",
      "field": "estado",
      "current": "Pending",
      "computed": "Completado",
      "evidence": {
        "children_completado": 5,
        "children_total": 5,
        "presence": 1.0
      },
      "target": "data"
    }
  ]
}
```

Key design decisions:
- `target: "schema"` → modifies `.stem` files
- `target: "data"` → modifies markdown files
- `target: "info"` → informational, no action unless user decides
- Evidence is always porcentual/numerical — never qualitative ("high", "medium")

### Commands

```bash
rootline analyze [path]                    # generates report (JSON by default)
rootline analyze [path] --output table     # human-readable summary
rootline analyze [path] --incremental      # delta against existing .stem
rootline apply report.json                 # applies approved inferences
```

---

## Part 6 — The Editable Report Cycle

### Current Flow (Disconnected)

```
init → (human edits .stem) → validate → (human corrects files) → fix
```

Each command operates independently. The output of `validate` doesn't feed back into `init`. The user manually bridges the gap.

### Proposed Flow (Unified via Report)

```bash
rootline analyze > report.json          # diagnose: schema + data issues
# human/agent edits report.json         # approve, reject, modify inferences
rootline apply report.json              # apply: modifies .stem AND/OR data
rootline analyze --incremental          # verify: re-diagnose post-apply
```

The report is the **single artifact** that flows through the cycle. It contains both schema proposals and data corrections, labeled by `target`.

### Direction of Correction

When `validate` finds an error today, it assumes the schema is correct and the data is wrong (→ `fix`). But sometimes the data is correct and the schema is outdated.

Example: 15 files have `tipo: helm-chart` which `validate` flags as enum violation. Is it more likely that 15 files are wrong, or that the enum is missing a value?

The report captures both possibilities:

```json
{"action": "correct_value", "target": "data", "evidence": {"files_affected": 15}},
{"action": "extend_enum", "target": "schema", "evidence": {"new_value": "helm-chart", "occurrences": 15}}
```

The consumer chooses which to approve. A skill could apply heuristics: "if >5 files have the same 'invalid' value, prefer `extend_enum` over `correct_value`."

### Relationship with Existing Commands

| Current command | Role in new flow |
|----------------|-----------------|
| `init` | Subsumed by `analyze` for schema generation |
| `validate` | Subsumed by `analyze` for error detection |
| `fix` | Subsumed by `apply` for data corrections |
| `migrate` | Partially subsumed: detection moves to `analyze --incremental`; transformations (rename, split) remain in `migrate` |
| `analyze` (new) | Unified diagnosis: schema inference + validation + delta detection |
| `apply` (new) | Unified application: schema changes + data corrections from report |

Existing commands continue working for direct use. `analyze`/`apply` are the unified flow for agents and advanced workflows.

---

## Part 7 — Incremental Inference

### The Problem

Today `init` assumes no `.stem` exists — it generates from scratch. When the schema already exists and data evolves, there's no way to ask "what changed?"

### Delta Detection

`analyze --incremental` loads the existing `.stem`, runs inference on current data, and reports the delta:

| Delta type | Example | Current coverage |
|-----------|---------|-----------------|
| New field in data | "3 tasks have `priority` not in schema" | Not detected |
| New enum value | "`tipo` has value `helm-chart` not in `.stem` values" | Detected by `validate` as error, not as schema suggestion |
| Obsolete field | "`cliente` in schema but present in 0/375 files" | Not detected |
| Type drift | "Field `ejecutable_en` declared as `string` but all values are single enum" | Not detected |
| Structural change | "New directory `F12-*` detected, no index file" | Not detected |

### Relationship with Migrate

`internal/migrate/` already has diff detection infrastructure (`field added/removed`, `type changed`, `enum changed`). But it compares `.stem` old vs `.stem` new — two schemas.

Incremental inference compares `.stem` vs data — schema against reality. The diff direction is inverted:

| Tool | Compares | Direction |
|------|---------|-----------|
| `migrate` | schema v1 vs schema v2 | schema → schema |
| `analyze --incremental` | schema vs data | schema → reality |

Migrate retains value for intentional transformations (bulk rename, split flat→hierarchical). Detection moves to `analyze`.

---

## Part 8 — Open Questions

### Q1: Report schema versioning

The report format (`"version": 1`) will evolve as new inference categories are added. How to handle forward compatibility? Options: strict versioning, or additive-only (new categories don't break old consumers).

### Q2: Agent architecture

Should semantic inference (categories 9, 11, 12, 13) use a single generalist agent or multiple specialized agents? Trade-off: single agent has full context but may conflate concerns; multiple agents are focused but require orchestration.

### Q3: Threshold configurability

Heuristic thresholds (80% for required, 50% for enum, etc.) are hardcoded in `internal/infer/infer.go`. Should they be configurable per-project? A homeserver project with 375 files has different statistical significance than a project with 10 files.

### Q4: Connection to v3 entity model

The [Intrinsic Hierarchy Principle](intrinsic-hierarchy-principle.md) proposes a v3 format with `entities:` and `index:` semantics. Several inference categories (4: structural, 7: back-references, 6: body structure) would benefit from entity-aware inference. Should `analyze` target v2 `.stem` only, or also support v3 output?

### Q5: Body content as first-class data

Categories 6-13 operate on body content, which rootline currently doesn't validate. Extending inference to body structure (required headings, section ordering) would require a new validation dimension beyond frontmatter. Is this in scope for rootline core, or should it remain in the skill/agent layer?

---

## Implementation Status (assessed 2026-03-20)

12 of 13 inference categories are implemented in `internal/infer/`. The analyze/apply pipeline is production-ready.

| # | Category | Status | Implementation |
|---|----------|--------|----------------|
| 1 | Schema (types, enums, required) | **Done** | `infer.go` |
| 2 | Sequences (prefix, digits) | **Done** | `hierarchy.go` |
| 3 | Aggregates (enum rollup) | **Done** | `internal/derive/` + `internal/migrate/aggregate.go` |
| 4 | Structural (require_index, min_children) | **Pending** | Not implemented — only category remaining |
| 5 | Links (allowed types) | **Done** | `link_validation.go` |
| 6 | Body structure (required sections) | **Done** | `body_sections.go` |
| 7 | Back-references | **Done** | `back_references.go` |
| 8 | Constants | **Done** | `constant_fields.go` |
| 9 | Heterogeneous dependencies | **Done** | `formal_dependency.go` (formal=engine, informal=agent-required) |
| 10 | Cross-epic references | **Done** | `cross_references.go` |
| 11 | Traceability (Contribuye a) | **Done** | `traceability_links.go` (verified=engine, unverified=agent-required) |
| 12 | Invariants (Preserva) | **Done** | `invariant_extraction.go` |
| 13 | Sub-schema by type | **Done** | `subschema_detection.go` |

**Three-layer model**: Realized as engine (11 categories deterministic) + agent-required flag (2 types: `informal_dependency_candidate`, `unverified_traceability`). Skills layer not formalized as separate code — skills operate via Claude Code plugin.

**Commands**: `rootline analyze` (12 detectors, `--incremental`, JSON/table output) and `rootline apply` (`--dry-run`, schema + data corrections) are fully functional.

**Test coverage**: `internal/infer/` at 97.8%, `internal/graph/` at 95.1%. E2E tests cover full analyze→apply pipeline.

**Remaining work**: Category 4 (structural rules inference) is the only gap. Low urgency — structural rules can be authored manually.

---

## References

### Rootline Source Files

- `cmd/rootline/init.go` — CLI command, mode selection
- `internal/infer/infer.go` — `Analyze()`, field stats, type inference
- `internal/infer/hierarchy.go` — `AnalyzeHierarchy()`, `DetectLevels()`, level detection
- `internal/migrate/aggregate.go` — `GenerateAggregates()`, semantic classification
- `internal/extract/` — Frontmatter extraction, wiki-link parsing from body
- `internal/rules/` — Validation engine, `.stem` loading, merge semantics
- `internal/proposal/` — Fix proposals including `extend_enum`
- `internal/migrate/` — Schema diff detection, bulk rename

### External References

- "Claude Skills and Subagents: Escaping the Prompt Engineering Hamster Wheel" — Towards Data Science, 2025. Architectural pattern for distributing work between deterministic skills and reasoning agents.
- [Intrinsic Hierarchy Principle](intrinsic-hierarchy-principle.md) — Rootline research on v2→v3 schema evolution and automatic vertical consistency.
- [Opportunity Areas](opportunity-areas.md) — Deferred feature ideas including Schema Registry.
