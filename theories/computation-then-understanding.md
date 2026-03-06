---
tipo: theory
confianza: emergent
fecha: "2026-03-03"
linea_origen: inference-engine-architecture
---
# Theory: Computation-then-Understanding

**Date documented:** 2026-03-03
**Confidence:** [x] Emergent | [ ] Developing | [ ] Consolidated

---

## Origin

- Line: [[closed/inference-engine-architecture]]
- Relevant cycles: Cycle 1 (category analysis), Cycle 2 (engine scope), corrected after user review

---

## The pattern or principle

Every inference category in rootline decomposes into engine computation (Go code) and optional agent reasoning (LLM). There is no intermediate layer.

> "If it can be expressed as regex, parsing, or counting, it's engine work — regardless of how complex the procedure looks."

### Expanded description

The research document proposed three layers: engine (deterministic), skills (codified procedures), and agents (semantic reasoning). Investigation revealed this trichotomy is false. Skills are how an agent organizes its work — they are agent behavior, not independent computation. A "skill" that consists of regex extraction, YAML parsing, and frequency counting is Go code that hasn't been written yet, not a separate architectural layer.

The correct decomposition is binary:
- **Engine**: Everything expressible as Go code. Regex, statistics, filesystem operations, threshold heuristics.
- **Agent**: Everything requiring language understanding. Semantic matching, disambiguation, redundancy detection.

Most inference categories (1-8, plus ~80% of 9-13) are fully engine-computable. The agent handles only the genuinely ambiguous residue (~20% of categories 9/11/12/13).

---

## Evidence

1. **Body content processing**: The engine already processes body content in 6 packages (extract, graph, query, proposal, derive, rules). The "frontmatter-only" characterization was factually incorrect.

2. **Category 13 (sub-schema by type)**: Described in research as requiring "program synthesis." Reality: `fieldsCompatible()` + `requires` rules already implement ~80%. The remaining work is YAML block extraction (regex) + co-occurrence statistics — both Go code.

3. **Category 12 (invariants)**: Described as requiring "reasoning." Reality: ~90% is `INV\d+:` regex extraction. Only redundancy detection between invariants needs LLM.

4. **"Skill" inflation**: Original analysis categorized deterministic procedures as "skill work" (~60-90% per category). User correction revealed these are engine-computable — the "skill" label obscured that regex + parsing + counting = Go code.

---

## Conditions of application

### Applies when:

- Evaluating where to implement a new inference category (engine vs agent)
- Assessing whether a "complex procedure" needs LLM or can be Go code
- Deciding between single vs multiple agents for a domain

### Does NOT apply when:

- The task genuinely requires language understanding (semantic similarity, natural language disambiguation)
- The input has no structural patterns to extract (purely unstructured prose)
- The domain lacks established conventions (no regex-matchable patterns)

---

## Connections

### With other theories in the system:

(none yet — first theory)

### With external knowledge:

- "Claude Skills and Subagents: Escaping the Prompt Engineering Hamster Wheel" (Towards Data Science, 2025) — argues for codifying procedures as skills. This theory refines the argument: codified procedures that are deterministic should be engine code, not LLM-executed skills.

---

## Artifacts that materialize this theory

| Artifact | Location | How it materializes it |
|----------|----------|----------------------|
| Corrected category proportions | closed/inference-engine-architecture/FIELD-LOG.md | Binary engine/agent split per category |
| Decisions Q1-Q5 | closed/inference-engine-architecture/CLOSURE.md | Architecture shaped by this theory |

---

## Open questions

- Does this theory hold for domains beyond rootline? (e.g., would a different file-based system have more genuinely semantic categories?)
- At what point does "complex regex + heuristics" become better served by an LLM than by Go code? (diminishing returns threshold)

---

## Evolution history

| Date | Change | Why |
|------|--------|-----|
| 2026-03-03 | Initial documentation | Emerged from line inference-engine-architecture, corrected after "skill layer" error |

---

*Emergent theory — R&D Framework*
