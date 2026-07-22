---
estado: Obsolete
tipo: outcome
---
# O02: Design Pi extension architecture

## Objetivo

Define the internal architecture for a maintainable Pi extension that shells out to Rootline safely, parses JSON consistently, and keeps read-only and mutating behavior separated.

## Criterios de Éxito

- CE1: Tool schemas and shared execution helpers are specified before implementation.
  - Verificar: Review T001 and T002 outputs.
- CE2: The architecture has explicit boundaries for CLI execution, parsing, rendering, permissions, and package resources.
  - Verificar: Review T005 architecture decision notes.

## Invariantes

- INV1: The extension treats Rootline CLI JSON as the integration boundary; it does not import Go internal packages.
  - Verificar: Check architecture docs and implementation tasks.

## Alcance

**In**:
- Tool schema design
- Shared CLI runner design
- Error/truncation design
- Permission boundary design

**Out**:
- Implementing the package skeleton
- Adding mutation tools

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-define-read-only-tool-schemas.md) | Define parameter and result contracts for read-only Rootline tools. |
| [T002](T002-design-rootline-cli-runner.md) | Design a shared CLI runner for executing rootline from Pi. |
| [T003](T003-define-output-truncation-and-rendering.md) | Define output truncation and optional TUI rendering behavior. |
| [T004](T004-design-permission-model.md) | Design read-only versus mutating tool activation and confirmations. |
| [T005](T005-write-extension-architecture-decision.md) | Write the architecture decision record for the Pi Rootline extension. |
