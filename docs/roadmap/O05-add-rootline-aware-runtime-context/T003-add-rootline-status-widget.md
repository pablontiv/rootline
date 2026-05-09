---
estado: Specified
tipo: task
---
# T003: Add status or widget showing Rootline project health.

**Outcome**: [O05 Add Rootline-aware runtime context](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T001-detect-rootline-project-state.md]]

## Preserva

- INV1: Injected context must be compact and derived from current repo state, not stale cached assumptions.
  - Verificar: Inspect before_agent_start and session_start behavior.

## Contexto

Esta task forma parte de O05 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Add status or widget showing Rootline project health.

## Alcance

**In**:
1. Status shows missing binary, no .stem, valid, or errors.
2. Widget output remains concise and non-blocking.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-detect-rootline-project-state.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Status shows missing binary, no .stem, valid, or errors.
- Widget output remains concise and non-blocking.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Pi UI status/widget docs`
