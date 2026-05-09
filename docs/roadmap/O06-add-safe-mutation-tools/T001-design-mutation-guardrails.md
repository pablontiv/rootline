---
estado: Completed
tipo: task
---
# T001: Design guardrails for mutating Rootline tools.

**Outcome**: [O06 Add safe mutation tools](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:../O05-add-rootline-aware-runtime-context/T005-test-context-in-rootline-and-plain-repos.md]]

## Preserva

- INV1: Mutating tools must not bypass Rootline validation or write outside user-approved paths.
  - Verificar: Review tool implementation and tests.

## Contexto

Esta task forma parte de O06 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Design guardrails for mutating Rootline tools.

## Alcance

**In**:
1. Guardrails define path restrictions, confirmation rules, and validation behavior.
2. Guardrails distinguish interactive and non-interactive Pi modes.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `../O05-add-rootline-aware-runtime-context/T005-test-context-in-rootline-and-plain-repos.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Guardrails define path restrictions, confirmation rules, and validation behavior.
- Guardrails distinguish interactive and non-interactive Pi modes.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/extensions/`
- `Pi tool_call docs`
