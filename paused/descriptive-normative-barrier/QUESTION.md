# Descriptive-Normative Barrier

## Central question

Where is the actual boundary between what the rootline engine can infer normatively (using computation) and what requires an LLM?

## Why does this matter?

The inference engine research document (Part 4) claims a hard barrier: "init can discover the descriptive schema (what exists). It cannot discover the normative schema (what should exist)." But preliminary investigation found 13 places where the engine already does normative inference. If the barrier is softer than claimed, the roadmap can assign more normative categories to the engine, reducing the agent's scope further.

## What kind of understanding am I seeking?

- [ ] **Describe** — What is this like? What characteristics does it have?
- [x] **Understand** — Why does it work this way? What does it mean?
- [ ] **Explore** — What's here? What happens if...?
- [ ] **Build** — Can I make something that...? (building is the method)

## Connections

- `[[closed/inference-engine-architecture]]` — previous line that resolved Q1-Q5 but didn't investigate this barrier
- `[[intake/inference-engine-architecture]]` — Part 4 makes the claim
- `[[theories/computation-then-understanding]]` — this line may refine or extend the theory

## Initial context

**What I already know:**
- The engine performs normative inference in 13 places across 6 files
- DetectDrift() says "parent SHOULD match children" (unanimous agreement)
- Proposal engine suggests schema extensions and data corrections
- Sibling inference proposes values based on 60%+ majority
- Infer.Analyze() decides "80% presence = required" — a normative threshold
- Stem health checks enforce 8 schema-level invariants

**What I think I know (unvalidated):**
- The real barrier is statistical vs semantic, not descriptive vs normative
- The gradient from statistical to semantic is clean
- All 13 existing normative cases are based on quantifiable signals

## Explicit presuppositions

| What I expect to find | Experience that influences this |
|-----------------------|-------------------------------|
| The barrier is statistical/semantic, not descriptive/normative | Found 13 normative cases, all use quantifiable signals |
| DetectDrift() is genuinely normative | It says "parent SHOULD match children" — that's a judgment |
| The gradient is clean | Haven't found intermediate cases yet |
| Statistical signals are always sufficient for normative inference | May be cases where statistics mislead (small samples, coincidental patterns) |

## Alternative hypotheses

A) **Statistical/semantic barrier is correct** — all engine-computable normative inference uses quantifiable signals (frequency, unanimity, distance). Semantic reasoning begins where statistics end.

B) **The barrier is softer: structural/semantic** — some normative inference is neither statistical nor semantic but structural (filesystem topology, directory depth, naming conventions). The gradient has three zones: statistical → structural → semantic.

C) **There is no barrier, only confidence levels** — the engine can do normative inference with varying confidence. "80% presence = required" is high confidence. "Parent SHOULD match children" is medium. What the research calls "needs LLM" is just low-confidence inference that the engine could still do with more sophisticated heuristics.

## Reformulated question (observational)

**Original:** Where is the actual boundary between engine-computable normative inference and LLM-required reasoning?

**Reformulated:** When I classify each of the 13 existing normative inference cases by signal type, what patterns emerge? Are there cases where the signal type doesn't cleanly predict whether the inference is engine-computable?

## Hypothesis validation

| Observation | Supports A | Supports B | Supports C | Notes |
|-------------|-----------|-----------|-----------|-------|
| DetectDrift() uses unanimous agreement (statistical) | ✓ | | | Clear statistical signal |
| Stem health checks use structural rules (not statistical) | | ✓ structural zone exists | | "Child SHOULD NOT redefine parent type" is structure, not statistics |
| Infer.Analyze() 80% threshold can mislead on small samples | | | ✓ confidence, not boundary | 4/5 files = 80% but not meaningful |
| **Cycle 1**: 13 cases group into 4 signal types, all engine-computable | ✗ not just statistical | ✓ partially — structural is distinct | ✗ there IS a hard boundary | Form vs meaning is the real distinction |
| **Cycle 1**: LLM-required cases all need meaning equivalence | ✓ boundary exists | ✓ boundary exists | ✗ not confidence — capability limit | Engine cannot determine two sentences mean the same thing |
| **Cycle 1**: 3 signal zones are within engine side, not spanning boundary | | ✗ 3 zones don't span the boundary | | Statistical, relational, structural are all "form" |

## Phases

| Phase | Cycles | Description | Status |
|-------|--------|-------------|--------|
| Explore | — | Classify 13 normative cases, map the gradient | Active |

## Status

- [ ] In backlog (unexplored)
- [x] In exploration (active cycles)
- [ ] Paused
- [ ] Closed

---

## Question evolution

| Date | Reformulated question | Why it changed |
|------|----------------------|----------------|
| 2026-03-03 | Initial formulation | Emerged from inference-engine-architecture closure |

---

*Line of inquiry — R&D Framework*
