---
estado: Obsolete
tipo: outcome
---
# O01: Map Rootline integration surface

## Objetivo

Produce a complete, evidence-backed map of Rootline capabilities and decide how each capability should appear in Pi: tool, slash command, prompt, context rule, or intentionally unsupported surface.

## Criterios de Éxito

- CE1: Every current Rootline CLI command is classified with integration mode, risk level, and expected JSON contract.
  - Verificar: Review docs/roadmap/O01-map-rootline-integration-surface/T001-inventory-rootline-commands.md and run rootline validate --all docs/roadmap/.
- CE2: The integration exposes only well-understood surfaces in later Outcomes.
  - Verificar: Confirm T003 contains an approved command-to-Pi mapping.

## Invariantes

- INV1: No Pi tool is implemented before its Rootline command contract and risk class are documented.
  - Verificar: Check downstream tasks reference this Outcome as source of truth.

## Alcance

**In**:
- CLI command inventory
- JSON contract inventory
- Read/analyze/mutation risk classification
- Decision matrix for Pi exposure

**Out**:
- Writing TypeScript extension code
- Publishing packages
- Changing Rootline command semantics

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-inventory-rootline-commands.md) | Inventory every Rootline CLI command and its flags that matter to Pi. |
| [T002](T002-capture-json-contracts.md) | Capture JSON output contracts for commands that Pi can consume. |
| [T003](T003-classify-pi-exposure.md) | Classify each command as Pi tool, slash command, prompt, context rule, or unsupported. |
| [T004](T004-define-compatibility-policy.md) | Define compatibility expectations between rootline CLI versions and the Pi extension. |
