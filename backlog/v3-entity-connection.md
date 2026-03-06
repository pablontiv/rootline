---
tipo: backlog-question
tema: schema, versioning, compatibility
fuente: intake/inference-engine-architecture
---
# Q4: Connection to v3 Entity Model

## Question

Should `analyze` target v2 `.stem` only, or also support v3 output?

## Source

`[[intake/inference-engine-architecture]]` — Part 8, Q4

## Context

The [Intrinsic Hierarchy Principle](../docs/research/intrinsic-hierarchy-principle.md) proposes a v3 format with `entities:` and `index:` semantics. Categories 4 (structural), 7 (back-references), and 6 (body structure) would benefit from entity-aware inference. Building `analyze` for v2 only may require rework if v3 lands.

## Why it matters for roadmap

If v3 is in scope, `analyze` needs to output both v2 and v3 formats. If v2-only, the implementation is simpler but creates migration debt later.

## Topic

schema, versioning, compatibility
