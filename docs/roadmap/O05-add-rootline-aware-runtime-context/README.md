---
estado: Obsolete
tipo: outcome
---
# O05: Add Rootline-aware runtime context

## Objetivo

Let Pi recognize Rootline-governed repositories and proactively guide the agent toward Rootline tools, schemas, validation state, and field-aware authoring.

## Criterios de Éxito

- CE1: Pi detects .stem-governed projects and surfaces concise Rootline context.
  - Verificar: Start Pi in a Rootline repo and inspect injected context/status.
- CE2: Rootline-aware UI aids do not overload context or block non-Rootline work.
  - Verificar: Test in repos with and without .stem files.

## Invariantes

- INV1: Injected context must be compact and derived from current repo state, not stale cached assumptions.
  - Verificar: Inspect before_agent_start and session_start behavior.

## Alcance

**In**:
- Project detection
- Status/widget
- before_agent_start guidance
- Field autocomplete exploration

**Out**:
- Mutating tools
- Complex operation guardrails

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-detect-rootline-project-state.md) | Detect Rootline project state from cwd. |
| [T002](T002-inject-rootline-agent-guidance.md) | Inject compact Rootline guidance before agent starts. |
| [T003](T003-add-rootline-status-widget.md) | Add status or widget showing Rootline project health. |
| [T004](T004-prototype-field-autocomplete.md) | Prototype autocomplete for schema fields and record references. |
| [T005](T005-test-context-in-rootline-and-plain-repos.md) | Test runtime context behavior in Rootline and non-Rootline repos. |
