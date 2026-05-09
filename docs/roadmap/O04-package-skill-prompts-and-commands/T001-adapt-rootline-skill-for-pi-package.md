---
estado: Completed
tipo: task
---
# T001: Superseded packaged Rootline skill adaptation.

**Outcome**: [O04 Package skill, prompts, and command UX](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:../O03-build-read-only-pi-package-mvp/T001-create-pi-package-skeleton.md]]

## Preserva

- INV1: Tool descriptions and prompts instruct agents to prefer Rootline tools over manual grep/read for governed records.
  - Verificar: Inspect packaged tool descriptions and prompt content.

## Contexto

Esta task forma parte de O04 y originalmente adaptó el Rootline skill into the Pi package. This approach is now superseded: the Pi package should expose tools and prompts only, avoiding a duplicate `rootline` skill.

## Alcance

**In**:
1. No packaged Pi skill is registered with `name: rootline`.
2. Agents receive Rootline guidance through tool descriptions and prompt templates.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `../O03-build-read-only-pi-package-mvp/T001-create-pi-package-skeleton.md` está completada o su salida está disponible.

## Criterios de Aceptación

- `integrations/pi/package.json` does not declare a `skills` resource.
- Rootline guidance is available through packaged tools and prompts.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `.claude/skills/rootline/SKILL.md`
- `integrations/pi/extensions/`
- `integrations/pi/prompts/`
