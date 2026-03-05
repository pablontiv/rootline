# Closure: inference-engine-architecture

**Closure date:** 2026-03-03
**Total duration:** 1 session (3 cycles)
**Cycles completed:** 3

---

## Original question

What architectural decisions (layer boundaries, report format, scope of body-content analysis, agent topology) need to be resolved before the inference engine research can be decomposed into implementable features?

## Final question

Same as original — the question was answered, not mutated.

## Why close?

- [x] Saturation reached (no new insights emerging)
- [x] Question answered (even if partially)
- [ ] Energy exhausted
- [ ] The question turned out to be the wrong one
- [ ] Forked into another line
- [ ] Other: _____

---

## Emergent theory

### Discovered patterns or principles

1. **Computation-then-understanding:** Every inference category decomposes into engine computation (Go code) and optional agent reasoning (LLM). There is no intermediate "skill layer" — skills are how the agent organizes its work, not a separate computational actor. If a procedure consists of regex, parsing, and counting, it's engine work.

2. **The research overestimated agent scope:** Categories 9/11/12/13 (assigned to "agents") are ~80% engine-computable. The engine already processes body content in 6 packages. The boundary between "frontmatter-only engine" and "body-aware agent" does not exist in practice.

### Key understandings

- **Two layers, not three.** Engine (Go) and agent (LLM). The "skill layer" from the research is a false category.
- **Body content is engine territory.** 6 packages already process body (extract, graph, query, proposal, derive, rules). Adding categories 6/7/8 is incremental.
- **One agent, not multiple.** The semantic residue (~20% of 4 categories) does not justify specialized agents.
- **Q1 was not open.** 18+ existing JSON contracts already follow additive-only versioning.
- **Q3 is a design decision, not architecture.** Hardcoded thresholds + porcentual evidence in the report.
- **Q4 is deferred correctly.** v3 has its own unresolved research.

---

## Decisions summary

| Q | Decision | Basis |
|---|----------|-------|
| Q1 | Additive-only, `version: 1` + `kind` | 18+ existing contracts follow this pattern |
| Q2 | One agent | ~20% semantic residue across 4 categories |
| Q3 | Hardcoded, report exposes porcentual evidence | 8 thresholds, consumer decides |
| Q4 | v2 only, v3 is separate research | v3 has own open questions |
| Q5 | Body content belongs in engine | Already processed in 6 packages |

---

## Artifacts produced

| Artifact | Location | Reusable? |
|----------|----------|-----------|
| Corrected category proportions (engine vs agent) | FIELD-LOG.md Cycle 1 | Yes — input for roadmap decomposition |
| Categories 4/6/7/8 implementation assessment | FIELD-LOG.md Cycle 2 | Yes — effort estimates for roadmap |
| Threshold inventory (8 thresholds with locations) | FIELD-LOG.md Cycle 3 | Yes — reference for future configurability |
| Theory: computation-then-understanding | theories/computation-then-understanding.md | Yes |

## Artifacts to move to /shared

- [ ] Category proportions table → /shared/patterns/ (when needed)

---

## Open questions

- How exactly should the single agent interact with engine output? (deferred to roadmap — implementation detail)
- Should the `analyze` report format support streaming for large datasets? (emerged but not explored)

## New questions for the backlog

None — existing backlog entries (Q1-Q5) are resolved. Backlog can be cleaned up.

---

## Connections

- `[[intake/inference-engine-architecture]]` — source document whose open questions this line resolved
- `[[theories/computation-then-understanding]]` — theory that emerged from this line
- Related: intrinsic-hierarchy-principle research (Q4 deferred to that line)

---

## Meta-reflection

The initial analysis used "skill" as an intermediate category between engine and agent. This created a false trichotomy that inflated non-engine proportions and muddied the architecture. The correction came from the user pointing out that skills are agent behavior, not a layer. Lesson: when a category exists that's "neither X nor Y", question whether it's real or an artifact of the analysis framework.

---

*Closure document — R&D Framework*
