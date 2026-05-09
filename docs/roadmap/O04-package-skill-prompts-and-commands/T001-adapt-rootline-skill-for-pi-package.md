---
estado: Specified
tipo: task
---
# T001: Adapt the existing Rootline skill into the Pi package.

**Outcome**: [O04 Package skill, prompts, and command UX](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:../O03-build-read-only-pi-package-mvp/T001-create-pi-package-skeleton.md]]

## Preserva

- INV1: Skills and prompts instruct agents to prefer Rootline tools over manual grep/read for governed records.
  - Verificar: Inspect packaged skill and prompt content.

## Contexto

Esta task forma parte de O04 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Adapt the existing Rootline skill into the Pi package.

## Alcance

**In**:
1. Skill metadata loads in Pi.
2. Skill references packaged tools and avoids Claude-specific assumptions where possible.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `../O03-build-read-only-pi-package-mvp/T001-create-pi-package-skeleton.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Skill metadata loads in Pi.
- Skill references packaged tools and avoids Claude-specific assumptions where possible.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `.claude/skills/rootline/SKILL.md`
- `integrations/pi/skills/`
