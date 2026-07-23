---
estado: Completed
tipo: task
---
# T009: Write `docs/analyze.md` command documentation

**Contribuye a**: closing the documentation coverage gap found by the docs-northstar audit (2026-07-22) — `analyze` is the flagship inference command (14 detectors: 12 data + 2 governance) and the only major command without its own doc under `docs/`.

## Context

Every other major command (`validate`, `query`, `tree`, `stats`, `graph`, `fix`, `describe`, `explain`, `init`, `new`, `set`, `migrate`) has a `docs/<cmd>.md`. `analyze` is referenced from README Section 5 and the agent skill, but has no reference doc covering detectors, `--incremental`, report schema, or the analyze → `schema apply` / `repair apply` loop.

## Acceptance criteria

- `docs/analyze.md` exists and follows the claim-verification procedure: every flag, example, and output shape verified against `rootline analyze --help`, source, or real execution.
- Documents the 14 detectors (12 data + 2 governance), `--incremental`, and the report consumption paths (`schema apply --report`, `repair apply --report`).
- No pinned product versions; JSON examples match real output shapes.
