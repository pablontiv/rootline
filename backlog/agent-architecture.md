# Q2: Agent Architecture — Single Generalist vs Multiple Specialized

## Question

Should semantic inference (categories 9, 11, 12, 13) use a single generalist agent or multiple specialized agents?

## Source

`[[intake/inference-engine-architecture]]` — Part 8, Q2

## Context

Trade-off: single agent has full context but may conflate concerns; multiple agents are focused but require orchestration. Categories 9 (heterogeneous deps), 11 (traceability), 12 (invariants), and 13 (sub-schema by type) all require semantic reasoning but operate on different signals.

## Why it matters for roadmap

Determines whether the roadmap has 1 epic for "agent inference" or 4 separate features with orchestration overhead.

## Topic

architecture, agents, inference
