---
estado: Specified
tipo: task
---
# T001: Detect Rootline project state from cwd.

**Outcome**: [O05 Add Rootline-aware runtime context](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:../O04-package-skill-prompts-and-commands/T003-implement-rootline-doctor-command.md]]

## Preserva

- INV1: Injected context must be compact and derived from current repo state, not stale cached assumptions.
  - Verificar: Inspect before_agent_start and session_start behavior.

## Contexto

Esta task forma parte de O05 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Detect Rootline project state from cwd.

## Alcance

**In**:
1. Detection distinguishes no-rootline, rootline-binary-only, and .stem-governed project states.
2. Detection is cheap enough for startup/reload.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `../O04-package-skill-prompts-and-commands/T003-implement-rootline-doctor-command.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Detection distinguishes no-rootline, rootline-binary-only, and .stem-governed project states.
- Detection is cheap enough for startup/reload.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/extensions/`
