# System Map
> Last updated: 2026-03-03

## Structure

### /intake (Reference library)

| Document | Classification | Referenced by |
|----------|---------------|---------------|
| inference-engine-architecture.md | ROADMAP (open questions resolved via /discover) | closed: inference-engine-architecture, theory: computation-then-understanding |

### /backlog (Unexplored questions)

| Question | Topic | Status |
|----------|-------|--------|
| [Q2: Agent Architecture](backlog/agent-architecture.md) | architecture, agents, inference | Resolved — 1 agent |
| [Q5: Body Content Scope](backlog/body-content-scope.md) | architecture, scope, validation | Resolved — engine |
| [Q4: v3 Entity Connection](backlog/v3-entity-connection.md) | schema, versioning, compatibility | Resolved — v2 only |
| [Q3: Threshold Configurability](backlog/threshold-configurability.md) | configuration, heuristics, inference | Resolved — hardcoded |
| [Q1: Report Versioning](backlog/report-versioning.md) | api-design, versioning, compatibility | Resolved — additive-only |

### /lines (Active lines of inquiry)

| Line | Central question | Cycles | Status |
|------|-----------------|--------|--------|
| (none) | | | |

### /theories (Emergent theories)

| Theory | Confidence | Connections |
|--------|-----------|-------------|
| [computation-then-understanding](theories/computation-then-understanding.md) | Emergent | Originated from inference-engine-architecture line |

### /closed

| Line | Closure date | Cycles | Key outcome |
|------|-------------|--------|-------------|
| [inference-engine-architecture](closed/inference-engine-architecture/CLOSURE.md) | 2026-03-03 | 3 | 5 Q's resolved. Two layers (engine + agent), not three. Body content → engine. One agent for ~20% semantic residue. |

### /paused

| Line | Central question | Cycles | Reason |
|------|-----------------|--------|--------|
| [descriptive-normative-barrier](paused/descriptive-normative-barrier/QUESTION.md) | Where is the actual boundary between engine-computable normative inference and LLM-required reasoning? | 1 | Finding sufficient for roadmap. 3 emerging questions deferred. |

### /shared (Reusable artifacts)

- /code: [empty]
- /patterns: [empty]
- /templates: QUESTION.md, FIELD-LOG.md, CLOSURE.md, THEORY.md

## Known Connections

- All 5 backlog entries originated from `[[intake/inference-engine-architecture]]` Part 8 — all resolved
- `[[theories/computation-then-understanding]]` emerged from `[[closed/inference-engine-architecture]]`
- Q4 (v3 connection) deferred to intrinsic-hierarchy-principle research (`docs/research/intrinsic-hierarchy-principle.md`)

## Emergent Patterns

- **Computation-then-understanding**: formalized as theory. Binary decomposition: engine (Go) vs agent (LLM). No intermediate skill layer.

## Current Context

- **Active lines**: 0/3
- **Paused lines**: 1
- **Closed lines**: 1
- **Theories**: 1 (Emergent)
- **Backlog**: 5 questions (all resolved)
- **Last session**: 2026-03-03 — descriptive-normative-barrier paused after Cycle 1. Barrier is form/meaning. All open questions resolved. Ready for roadmap.
