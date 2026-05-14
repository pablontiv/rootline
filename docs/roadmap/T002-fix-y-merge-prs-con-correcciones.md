---
estado: Specified
tipo: task
---
# T002: Fix y merge de PR #29 (commit) y PR #30 (go.mod)

**Contribuye a**: Limpiar PRs con código válido pero con problemas formales

## Contexto

**PR #29 — splitDotPath optimization** (+3/-16 líneas, `cmd/rootline/validate.go`):
Reemplaza loop manual con `strings.FieldsFunc`. Benchmarks: 6.5x más rápido, 98.6% menos allocations.
El código es correcto, pero el commit message es un placeholder (`⚡ [performance improvement description]`),
violando el hook `.githooks/commit-msg` que exige formato conventional commits.
Fix: amend a `perf(validate): optimize splitDotPath with strings.FieldsFunc`.

**PR #30 — fix TODO placeholder en rules_test.go** (+3/-3 líneas):
El test fix es correcto (reemplaza `<!-- TODO -->` con texto descriptivo en español).
Pero incluye downgrade de go.mod de 1.25.0 a 1.24.3, violando el requisito Go 1.25+ del proyecto (CLAUDE.md).
Fix: revertir el hunk de go.mod antes de mergear.

## Alcance

**In**:
1. PR #29: checkout branch, amend commit message a `perf(validate): optimize splitDotPath with strings.FieldsFunc`, force-push, merge
2. PR #30: checkout branch, editar go.mod para restaurar `go 1.25.0`, commit adicional, merge

**Out**:
- No cambiar la lógica de código de ninguno de los dos PRs

## Estado inicial esperado

- PR #29 con commit message placeholder
- PR #30 con go.mod en 1.24.3

## Criterios de Aceptación

- PR #29 mergeado con commit message que pasa `grep "^perf(validate):" <(git -C /home/shared/rootline log --oneline -20)`
- PR #30 mergeado con `go 1.25.0` en go.mod de master
- `grep "^go " /home/shared/rootline/go.mod` retorna `go 1.25.0`

## Fuente de verdad

- `/home/shared/rootline/go.mod`
- `gh pr list --repo pablontiv/rootline --json number,title --state open`
