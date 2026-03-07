---
estado: Pending
---
# E14: Aggregate Consistency Engine

**Metrica de exito**: `rootline fix --all` resolves aggregate drift automatically; `rootline validate --all` shows 0 aggregate errors after fix
**Timeline**: 2026-Q1

## Intencion

Analysis of 200+ Claude Code sessions revealed that manual frontmatter propagation is the #1 friction pattern (26 sessions with direct evidence). When child documents change `estado`, parent READMEs retain stale values. The aggregate engine correctly computes the new value in-memory (`record.Derived`) but has no bridge to write it back to disk. Users manually edit 3+ files per completed task.

This Epic implements the missing bridge: a `PropagateAggregate` proposal type that compares computed aggregate values against stored frontmatter and generates proposals for the existing `fix` pipeline to apply. It also adds formula completeness diagnostics and post-merge automation.

## Postcondiciones

- P1: `rootline fix --all` writes computed aggregate values to index file frontmatter
- P2: Incomplete aggregate formulas produce stem-health warnings
- P3: Aggregate propagation runs automatically after git merge

## Invariantes

- INV1: Existing `fix --all` proposal types continue working unchanged
  - Verificar: `go test ./internal/fix/ ./internal/proposal/ -race`
- INV2: Propagation only affects index files with `aggregate:` definitions
- INV3: Propagation is skipped (with warning) when formula doesn't cover all descendant values

## Out of Scope

- v3 entity model changes (research blocked — circular tension unresolved)
- `derive:` field write-back (different semantics, write-loop risk)
- Per-file propagation opt-out annotations

## Features

| ID | Nombre | Descripcion |
|----|--------|-------------|
| F01 | [Propagate Aggregate Bridge](F01-propagate-aggregate-bridge/) | Core bridge between aggregate computation and disk write-back |
| F02 | [Formula Completeness Diagnostics](F02-formula-completeness-diagnostics/) | Stem-health warning for incomplete aggregate formulas |
| F03 | [Post-Merge Auto-Fix](F03-post-merge-auto-fix/) | Git hook for automatic aggregate propagation after merge |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | Core bridge — everything depends on this |
| F02 | F01 | Formula check supplements the propagation pre-check |
| F03 | F01 | Post-merge hook runs `fix --all` which needs propagation |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-03-06 | Default-on, not opt-in | H1: 4+ sessions show users expecting fix to handle aggregates; zero intentional overrides |
| 2026-03-06 | Formula pre-check before propagation | H2: 3 docs have Obsolete, formula produces Pending (incorrect) |
| 2026-03-06 | Design for v2, not v3 | H3: v3 blocked on circular tension; detector interface clean enough for later extension |
| 2026-03-06 | New PropagateAggregate type | Different semantics from CorrectValue; enables distinct reporting |
