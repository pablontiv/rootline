---
estado: Pending
tipo: feature
---
# F03: Post-Merge Auto-Fix

**Epic**: [E14 Aggregate Consistency Engine](../README.md)
**Satisface**: P3
**Objetivo**: Eliminate pre-push blocking by auto-propagating after merges
**Beneficio**: Worktree/subagent workflows no longer leave aggregate drift
**Milestone**: After merging a worktree branch, aggregates are consistent without manual intervention

## Scope

**In**: Post-merge git hook that runs `rootline fix --all`
**Out**: Pre-commit hooks, CI integration, watch mode

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Git Hook Integration](S001-git-hook-integration/) | Post-merge hook auto-propagates aggregates |

## Invariantes

- INV1 (heredado): `go test ./... -race` pasa verde

## Dependencias

- F01 must be complete first (hook runs `fix --all` which needs propagation)

## Fuente de verdad

- `.githooks/` — existing git hooks directory
- `cmd/rootline/fix.go` — fix --all command
