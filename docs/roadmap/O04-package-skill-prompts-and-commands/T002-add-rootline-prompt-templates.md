---
estado: In Progress
tipo: task
---
# T002: Add prompt templates for query, validate, analyze, and roadmap workflows.

**Outcome**: [O04 Package skill, prompts, and command UX](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T001-adapt-rootline-skill-for-pi-package.md]]

## Preserva

- INV1: Skills and prompts instruct agents to prefer Rootline tools over manual grep/read for governed records.
  - Verificar: Inspect packaged skill and prompt content.

## Contexto

Esta task forma parte de O04 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Add prompt templates for query, validate, analyze, and roadmap workflows.

## Alcance

**In**:
1. Prompts include descriptions and argument hints.
2. Prompts call out the appropriate Rootline tool or command workflow.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-adapt-rootline-skill-for-pi-package.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Prompts include descriptions and argument hints.
- Prompts call out the appropriate Rootline tool or command workflow.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/prompts/`
