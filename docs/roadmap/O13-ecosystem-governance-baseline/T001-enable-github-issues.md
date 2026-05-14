---
estado: Completed
tipo: task
---
# T001: Enable GitHub issues

**Contribuye a**: align rootline with the ecosystem standard where issues are open for public engagement (currently disabled — the only repo in the ecosystem with issues off).

## Alcance

**In**:
- Enable issues on pablontiv/rootline via GitHub API (`has_issues: true`)

**Out**:
- No file changes in the repo

## Criterios de Aceptación

- `gh repo view pablontiv/rootline --json hasIssuesEnabled` returns `{"hasIssuesEnabled":true}`

## Fuente de verdad

- GitHub API: PATCH repos/pablontiv/rootline
