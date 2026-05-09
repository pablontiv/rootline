---
estado: Pending
tipo: outcome
---
# O04: Package skill, prompts, and command UX

## Objetivo

Make the integration usable by humans and agents through packaged skills, prompt templates, and slash commands, not only raw tool calls.

## Criterios de Éxito

- CE1: The package includes Rootline skill and prompts that Pi discovers.
  - Verificar: Run pi config/list or headless Pi resource discovery.
- CE2: Slash commands cover common diagnostics and validation workflows.
  - Verificar: Run /rootline doctor and /rootline validate in a test session or equivalent command handler test.

## Invariantes

- INV1: Skills and prompts instruct agents to prefer Rootline tools over manual grep/read for governed records.
  - Verificar: Inspect packaged skill and prompt content.

## Alcance

**In**:
- Packaged Rootline skill
- Prompt templates
- Slash commands for doctor/validate/tree
- Usage docs

**Out**:
- Runtime context injection
- Mutating tools
- Package publication

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-adapt-rootline-skill-for-pi-package.md) | Adapt the existing Rootline skill into the Pi package. |
| [T002](T002-add-rootline-prompt-templates.md) | Add prompt templates for query, validate, analyze, and roadmap workflows. |
| [T003](T003-implement-rootline-doctor-command.md) | Implement /rootline doctor command. |
| [T004](T004-implement-validation-and-tree-commands.md) | Implement convenience slash commands for validation and tree inspection. |
| [T005](T005-document-local-command-usage.md) | Document local usage of skills, prompts, and commands. |
