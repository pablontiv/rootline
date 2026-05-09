---
estado: Pending
tipo: outcome
---
# O03: Build read-only Pi package MVP

## Objetivo

Create a local Pi package under integrations/pi that exposes Rootline read-only capabilities through native Pi tools backed by the Rootline CLI.

## Criterios de Éxito

- CE1: A Pi package skeleton exists and can be installed locally.
  - Verificar: Run pi install -l ./integrations/pi or equivalent local extension load.
- CE2: Read-only Rootline tools execute from Pi and return parsed JSON or clear errors.
  - Verificar: Run targeted headless Pi checks for rootline_query, rootline_validate, rootline_describe, rootline_tree, and rootline_stats.

## Invariantes

- INV1: MVP tools do not mutate repository files.
  - Verificar: Inspect extension tool implementations and tests.

## Alcance

**In**:
- integrations/pi package skeleton
- Read-only tools
- Shared CLI runner
- Local install validation

**Out**:
- Mutating tools
- Automatic context injection
- npm publication

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-create-pi-package-skeleton.md) | Create the integrations/pi package skeleton with manifest resources. |
| [T002](T002-implement-rootline-cli-runner.md) | Implement the shared rootline CLI runner in the extension package. |
| [T003](T003-implement-query-describe-tools.md) | Implement rootline_query and rootline_describe tools. |
| [T004](T004-implement-validate-tree-stats-tools.md) | Implement rootline_validate, rootline_tree, and rootline_stats tools. |
| [T005](T005-add-read-only-tool-tests.md) | Add tests or executable fixtures for read-only tools. |
| [T006](T006-verify-local-pi-install.md) | Verify the package locally through Pi. |
