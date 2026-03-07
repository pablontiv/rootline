---
estado: Completed
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

## Verification Results

**Procedure executed**: 2026-03-07

1. Created branch `test/post-merge-verify` from master
2. Marked T002 as Completed on the branch (S001, F03, E14 READMEs still `Pending`)
3. Merged branch back to master with `git merge test/post-merge-verify`
4. Post-merge hook fired: rebuilt rootline, then ran `rootline fix --all docs/epics/`
5. **Results**:
   - S001 README: `Pending` → `Completed` (both tasks completed)
   - F03 README: `Pending` → `Completed` (all stories completed)
   - E14 README: `Pending` → `Completed` (all features completed)
   - `rootline validate --all docs/epics/`: 0 errors across 574 files

All 3 acceptance criteria satisfied.

## Fuente de verdad

- `.githooks/post-merge` — post-merge hook script
- `cmd/rootline/fix.go` — fix --all with propagation
