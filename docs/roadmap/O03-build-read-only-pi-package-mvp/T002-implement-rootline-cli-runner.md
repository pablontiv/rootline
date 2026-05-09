---
estado: In Progress
tipo: task
---
# T002: Implement the shared rootline CLI runner in the extension package.

**Outcome**: [O03 Build read-only Pi package MVP](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T001-create-pi-package-skeleton.md]]

## Preserva

- INV1: MVP tools do not mutate repository files.
  - Verificar: Inspect extension tool implementations and tests.

## Contexto

Esta task forma parte de O03 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Implement the shared rootline CLI runner in the extension package.

## Alcance

**In**:
1. Runner supports cwd, abort signal, timeout, JSON parse, and structured errors.
2. Runner has focused unit or fixture coverage where practical.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-create-pi-package-skeleton.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Runner supports cwd, abort signal, timeout, JSON parse, and structured errors.
- Runner has focused unit or fixture coverage where practical.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/extensions/`
