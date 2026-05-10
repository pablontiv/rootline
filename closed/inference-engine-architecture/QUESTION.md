---
tipo: question
estado: closed
tipo_entendimiento: understand
fecha_inicio: "2026-03-03"
---
# Inference Engine Architecture — Pre-implementation Decisions

## Central question

What architectural decisions (layer boundaries, report format, scope of body-content analysis, agent topology) need to be resolved before the inference engine research can be decomposed into implementable features?

## Why does this matter?

The research document (`[[intake/inference-engine-architecture]]`) proposes a three-layer inference architecture (engine/skills/agents), a unified `analyze`/`apply` cycle, and 13 inference categories. It contains 5 explicit open questions and several untested presuppositions. Decomposing this into a roadmap without resolving these would create features that may need rearchitecting mid-implementation.

## What kind of understanding am I seeking?

- [ ] **Describe** — What is this like? What characteristics does it have?
- [x] **Understand** — Why does it work this way? What does it mean?
- [ ] **Explore** — What's here? What happens if...?
- [ ] **Build** — Can I make something that...? (building is the method)

## Connections

- `[[intake/inference-engine-architecture]]` — source document
- Related: [Intrinsic Hierarchy Principle](../../docs/research/intrinsic-hierarchy-principle.md) — v2→v3 schema model (Q4 depends on this)
- Related: [Opportunity Areas](../../docs/research/opportunity-areas.md) — deferred features context

## Initial context

**What I already know:**
- `rootline init` currently implements 3 of 13 inference categories (schema, sequences, aggregates)
- The Go engine handles deterministic computation; `internal/infer/` has `Analyze()` and `AnalyzeHierarchy()`
- `internal/migrate/` has diff detection infrastructure that could be reused for incremental inference
- The research was validated against a real dataset (375 files, 4 hierarchy levels)

**What I think I know (unvalidated):**
- The three-layer model (engine/skills/agents) has clean boundaries
- `analyze` can subsume `init` + `validate` without breaking existing UX
- 13 categories are exhaustive (derived from one dataset)
- The descriptive/normative barrier is a hard limit
- Skills = reproducible procedure vs Agents = semantic ambiguity is a clean distinction

## Explicit presuppositions

| What I expect to find | Experience that influences this |
|-----------------------|-------------------------------|
| Three layers (engine/skills/agents) are the right decomposition | Go engine already handles deterministic work well; LLMs handle ambiguity |
| A unified report format captures both schema and data issues | Existing `validate` and `fix` already produce structured output |
| 13 categories cover inference needs | Analysis of a single large dataset (homeserver) |
| Incremental inference is necessary and distinct from `migrate` | Users edit .stem manually; schema drifts from data |
| Body content analysis belongs outside the engine | Rootline currently only validates frontmatter |

## Alternative hypotheses

A) **Three-layer model is correct** — engine handles categories 1-8, skills handle 5/6/10, agents handle 9/11/12/13. Clean boundaries, each layer has a distinct nature.

B) **Two layers suffice** — engine handles all deterministic categories (1-8), and everything semantic goes through a single agent that uses skills as tools internally. The "skill layer" is an implementation detail of the agent, not an independent architectural layer.

C) **The layers are wrong** — categorization should be by data source (frontmatter vs body vs filesystem) rather than by reasoning nature (deterministic vs procedural vs semantic). This would group categories differently and may reveal simpler implementation paths.

## Reformulated question (observational)

**Original:** What architectural decisions are needed before the inference engine can be decomposed into implementable features?

**Reformulated:** When I attempt to implement the first new inference category (e.g., structural rules, category 4), what obstacles do I encounter that require decisions not covered by the research document? What assumptions in the research break when confronted with a second dataset?

## Hypothesis validation

| Observation | Supports A | Supports B | Supports C | Notes |
|-------------|-----------|-----------|-----------|-------|
| Category 5' (co-occurrence) moved from skill→engine in the research itself | ✓ boundaries shift | ✓ layer distinction is fuzzy | ✓ categorization is unstable | Document already shows a category migrating layers |
| `internal/migrate/` has diff detection reusable for incremental | ✓ engine handles more | | | Suggests engine scope may be broader than proposed |
| Body content extraction already exists in `internal/extract/` | | | ✓ data-source grouping may be natural | Engine already touches body content for wiki-links |
| **Cycle 1**: Engine processes body in 6 packages (extract, graph, query, proposal, derive, rules) | ✗ boundary is porous | ✓ engine already does body work | ✓ data-source distinction is artificial | Q5 presupposition ("body is outside engine") is false |
| **Cycle 1**: Categories 9/11/12/13 are ~80% engine-computable (Go code) | ✗ agent layer is much smaller | ✓ two layers, not three | | "Computation-then-understanding" pattern emerged. Skills are agent behavior, not a layer. |
| **Cycle 1**: `fieldsCompatible()` + `requires` rules already implement 80% of category 13 | ✗ less agent work needed | ✓ engine scope is broader | | Category 13 is mislabeled as "program synthesis" |
| **Cycle 2**: Category 4 (structural) already fully implemented with tests | ✗ engine already wider than proposed | ✓ engine handles more | | Research underestimated existing infrastructure |
| **Cycle 2**: Category 8 (constants) needs one conditional branch in `infer.go` | ✗ trivial addition | ✓ engine scope is natural | | Data already collected, just not flagged |
| **Cycle 2**: Categories 6/7 need only regex + frequency analysis on existing `Record.Body` | ✗ incremental, not architectural | ✓ body is already engine territory | ✗ data-source grouping not needed | Adding body analysis is extension, not new dimension |
| **Cycle 2**: All 4 categories are deterministic — no ambiguity, no reasoning | ✗ no agent work here | ✓ Q5 resolved: body → engine | | Q2 resolved: 1 agent for semantic residue only |
| **Cycle 3**: 18+ JSON contracts already use `version: 1` + `kind`, additive-only | | ✓ Q1 follows existing convention | | Q1 was not actually open — convention already existed |
| **Cycle 3**: 8 hardcoded thresholds, report exposes porcentual evidence | | ✓ hardcoded is correct default | | Q3: configurability is future work, not architectural |
| **Cycle 3**: v3 research has own open questions (children = re-declaration?) | | ✓ v2-only is correct scope | | Q4: v3 is separate line of inquiry |

## Phases

| Phase | Cycles | Description | Status |
|-------|--------|-------------|--------|
| Explore | — | Resolve Q1-Q5, validate presuppositions | Active |

## Status

- [ ] In backlog (unexplored)
- [x] In exploration (active cycles)
- [ ] Paused
- [ ] Closed

---

## Question evolution

| Date | Reformulated question | Why it changed |
|------|----------------------|----------------|
| 2026-03-03 | Initial formulation | Created from intake of inference-engine-architecture.md |

---

*Line of inquiry — R&D Framework*
