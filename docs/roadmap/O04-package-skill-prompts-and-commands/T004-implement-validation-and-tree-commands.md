---
estado: Specified
tipo: task
---
# T004: Implement convenience slash commands for validation and tree inspection.

**Outcome**: [O04 Package skill, prompts, and command UX](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T003-implement-rootline-doctor-command.md]]

## Preserva

- INV1: Skills and prompts instruct agents to prefer Rootline tools over manual grep/read for governed records.
  - Verificar: Inspect packaged skill and prompt content.

## Contexto

Esta task forma parte de O04 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Implement convenience slash commands for validation and tree inspection.

## Alcance

**In**:
1. /rootline validate runs the validation flow for a target path.
2. /rootline tree shows a concise project structure summary.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T003-implement-rootline-doctor-command.md` está completada o su salida está disponible.

## Criterios de Aceptación

- /rootline validate runs the validation flow for a target path.
- /rootline tree shows a concise project structure summary.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/extensions/`
