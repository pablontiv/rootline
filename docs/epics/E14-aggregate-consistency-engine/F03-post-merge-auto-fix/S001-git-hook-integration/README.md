---
estado: Pending
tipo: historia
---
# S001: Git Hook Integration

**Feature**: [F03 Post-Merge Auto-Fix](../README.md)
**Capacidad**: Post-merge hook auto-propagates aggregate values
**Cubre**: Automated post-merge propagation eliminates pre-push blocking

## Antes / Despues

**Antes**: After merging a worktree branch with completed tasks, aggregate drift blocks the pre-push hook. Users must run `rootline fix --all` manually before pushing. This was observed in 4 sessions (P4 pattern).

**Despues**: Post-merge hook runs `rootline fix --all` automatically. Pre-push validation passes without manual intervention.

## Criterios de Aceptacion (semanticos)

- [ ] `.githooks/post-merge` exists and runs `rootline fix --all`
- [ ] After merging a branch with completed tasks, parent READMEs are auto-updated
- [ ] Pre-push validation passes without manual intervention

## Invariantes

- INV1: Existing git hooks unchanged
  - Verificar: `.githooks/pre-commit` and `.githooks/pre-push` work as before

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-post-merge-hook.md) | Create .githooks/post-merge that runs rootline fix --all |
| [T002](T002-verify-post-merge-flow.md) | Test: merge worktree, verify aggregates propagated |

## Fuente de verdad

- `.githooks/` — existing pre-commit, pre-push, commit-msg hooks
- `cmd/rootline/fix.go` — fix --all command
