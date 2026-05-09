---
estado: In Progress
tipo: task
---
# T001: Create the integrations/pi package skeleton with manifest resources.

**Outcome**: [O03 Build read-only Pi package MVP](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:../O02-design-pi-extension-architecture/T005-write-extension-architecture-decision.md]]

## Preserva

- INV1: MVP tools do not mutate repository files.
  - Verificar: Inspect extension tool implementations and tests.

## Contexto

Esta task forma parte de O03 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Create the integrations/pi package skeleton with manifest resources.

## Alcance

**In**:
1. integrations/pi/package.json declares pi extensions, skills, and prompts locations.
2. Peer dependencies follow Pi package guidance.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `../O02-design-pi-extension-architecture/T005-write-extension-architecture-decision.md` está completada o su salida está disponible.

## Criterios de Aceptación

- integrations/pi/package.json declares pi extensions, skills, and prompts locations.
- Peer dependencies follow Pi package guidance.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/`
- `Pi packages docs`
