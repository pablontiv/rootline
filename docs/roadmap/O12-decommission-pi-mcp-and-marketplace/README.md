---
tipo: outcome
---
# Decommission Pi, MCP, and marketplace publishing

Rootline stops owning Pi integration packaging, removes MCP support, and removes active Claude marketplace publication while preserving local Claude skill assets.

## Criterios de Aceptación

- Rootline no longer contains an active integrations/pi package or CI gate.
- Rootline no longer exposes MCP support or the rootline serve command.
- Rootline no longer publishes local Claude skills to an external marketplace.
- Active Rootline docs and agent guidance no longer advertise removed product surfaces.
- Rootline validation and tests pass after the removals.

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-remove-rootline-mcp-support.md) | Delete the MCP server surface from Rootline, including the serve command, internal MCP package, tests, dependency, and active documentation references. |
| [T002](T002-remove-rootline-pi-package-and-ci-gate.md) | Remove the co-located Pi package from Rootline after its ownership moves to Pinata, and remove Rootline CI/release coupling for that package. |
| [T003](T003-remove-claude-marketplace-publishing.md) | Remove the active workflow and documentation that publish Rootline Claude skills to an external marketplace, while keeping local .claude skill assets in the repository. |
| [T004](T004-clean-active-rootline-docs-and-agent-contracts.md) | Update active Rootline documentation and agent guidance so they match the new product boundaries after Pi package, MCP, and marketplace publishing are removed. |
