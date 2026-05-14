---
estado: Specified
tipo: task
---
# T001: Merge de los 4 PRs listos para merge

**Contribuye a**: Limpiar PRs pendientes con CI verde o sin riesgo

## Contexto

4 PRs listos para merge en rootline:
- PR #24: goldmark 1.7.17→1.8.2 (CI verde)
- PR #26: docs — Reddit communities launch checklist (CI verde, docs only)
- PR #27: RenderHTML tests + refactor (agente: alta calidad, tests pasan, bajo riesgo)
- PR #31: actions/setup-go 6.3→6.4 (MERGEABLE, solo workflow)

## Alcance

**In**:
1. `gh pr merge 24 --repo pablontiv/rootline --merge`
2. `gh pr merge 26 --repo pablontiv/rootline --merge`
3. `gh pr merge 27 --repo pablontiv/rootline --merge`
4. `gh pr merge 31 --repo pablontiv/rootline --merge`
5. `git -C /home/shared/rootline pull --rebase origin master`

**Out**:
- No modificar código más allá del merge

## Estado inicial esperado

- PRs #24, #26, #27, #31 abiertos

## Criterios de Aceptación

- Los 4 PRs no aparecen en `gh pr list --repo pablontiv/rootline --state open`
- `git -C /home/shared/rootline log --oneline -6` muestra los merges

## Fuente de verdad

- `gh pr list --repo pablontiv/rootline --state open --json number,title`
