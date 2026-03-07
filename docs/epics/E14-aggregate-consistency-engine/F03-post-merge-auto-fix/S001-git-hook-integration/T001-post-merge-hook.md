---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Create post-merge hook for aggregate propagation

**Story**: [S001 Git Hook Integration](README.md)
**Contribuye a**: .githooks/post-merge exists and runs rootline fix --all

## Preserva

- INV1: Existing git hooks unchanged
  - Verificar: `.githooks/pre-commit` and `.githooks/pre-push` work as before

## Contexto

The project uses `.githooks/` for git hooks (configured via `git config core.hooksPath .githooks`). After merging worktree branches where subagents completed tasks, aggregate values in parent READMEs are stale. A post-merge hook that runs `rootline fix --all` on the docs/epics/ directory automatically propagates aggregate values before the user attempts to push.

## Alcance

**In**:
1. Create `.githooks/post-merge` shell script
2. Script runs `rootline fix --all docs/epics/ 2>/dev/null || true` (silent, non-blocking)
3. Script should check if `rootline` is available before running
4. Make script executable (`chmod +x`)

**Out**: Pre-commit hooks, CI integration, rootline watch mode

## Estado inicial esperado

- `.githooks/` directory exists with pre-commit, pre-push, commit-msg hooks
- `git config core.hooksPath` is set to `.githooks`
- F01 completed: `fix --all` propagates aggregates by default

## Criterios de Aceptacion

- `.githooks/post-merge` exists and is executable
- Script runs `rootline fix --all docs/epics/` when rootline is available
- Script is silent and non-blocking (exit 0 regardless)
- Existing hooks are unmodified

## Fuente de verdad

- `.githooks/` — existing hooks (pre-commit, pre-push, commit-msg)
