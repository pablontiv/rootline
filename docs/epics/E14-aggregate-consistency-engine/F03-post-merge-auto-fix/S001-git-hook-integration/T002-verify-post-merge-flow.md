---
estado: Pending
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Verify post-merge aggregate propagation flow

**Story**: [S001 Git Hook Integration](README.md)
**Contribuye a**: After merging a branch, parent READMEs are auto-updated; pre-push passes

[[blocks:T001-post-merge-hook]]

## Preserva

- INV1: Existing git hooks unchanged
  - Verificar: `.githooks/pre-commit` and `.githooks/pre-push` work as before

## Contexto

End-to-end verification that the post-merge hook works in a realistic workflow: create a worktree, complete tasks (mark estado: Completed), merge back to main branch, and verify that the post-merge hook automatically propagates aggregate values to parent READMEs.

## Alcance

**In**:
1. Manual verification procedure (not automated test — git hooks are hard to test in CI):
   - Create a test branch and worktree
   - Mark a task as Completed in the worktree
   - Merge the branch back
   - Verify that parent README estado was updated by the post-merge hook
   - Run `rootline validate --all docs/epics/` to confirm 0 errors
2. Document the verification procedure in this file's results

**Out**: Automated CI test for post-merge hook, worktree management

## Estado inicial esperado

- T001 completed: .githooks/post-merge exists
- F01 completed: fix --all propagates aggregates
- A stale aggregate exists or can be created for testing

## Criterios de Aceptacion

- Post-merge hook fires after `git merge`
- Parent README estado is updated after merge
- `rootline validate --all docs/epics/` shows 0 aggregate errors after merge

## Fuente de verdad

- `.githooks/post-merge` — post-merge hook script
- `cmd/rootline/fix.go` — fix --all with propagation
