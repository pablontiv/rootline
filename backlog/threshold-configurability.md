---
tipo: backlog-question
tema: configuration, heuristics, inference
fuente: intake/inference-engine-architecture
---
# Q3: Threshold Configurability

## Question

Should inference heuristic thresholds (80% for required, 50% for enum, etc.) be configurable per-project?

## Source

`[[intake/inference-engine-architecture]]` — Part 8, Q3

## Context

Thresholds are hardcoded in `internal/infer/infer.go`. A project with 375 files has different statistical significance than one with 10 files. Options: keep hardcoded (simple), make configurable via `.stem` or CLI flags (flexible), or auto-adjust based on sample size (smart but complex).

## Why it matters for roadmap

If configurable, `analyze` needs a config spec and CLI flags. If auto-adjusted, the inference engine needs statistical significance logic. If hardcoded, no additional work.

## Topic

configuration, heuristics, inference
