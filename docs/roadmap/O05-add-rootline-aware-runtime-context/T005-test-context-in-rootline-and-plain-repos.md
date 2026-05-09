---
estado: Completed
tipo: task
---
# T005: Test runtime context behavior in Rootline and non-Rootline repos.

**Outcome**: [O05 Add Rootline-aware runtime context](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T004-prototype-field-autocomplete.md]]

## Preserva

- INV1: Injected context must be compact and derived from current repo state, not stale cached assumptions.
  - Verificar: Inspect before_agent_start and session_start behavior.

## Contexto

Esta task forma parte de O05 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Test runtime context behavior in Rootline and non-Rootline repos.

## Alcance

**In**:
1. Rootline repos receive guidance and status.
2. Plain repos do not receive irrelevant Rootline instructions.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T004-prototype-field-autocomplete.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Rootline repos receive guidance and status.
- Plain repos do not receive irrelevant Rootline instructions.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi tests`
- `manual Pi sessions`
