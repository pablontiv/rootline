---
estado: Completed
tipo: task
---
# T002: Inject compact Rootline guidance before agent starts.

**Outcome**: [O05 Add Rootline-aware runtime context](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T001-detect-rootline-project-state.md]]

## Preserva

- INV1: Injected context must be compact and derived from current repo state, not stale cached assumptions.
  - Verificar: Inspect before_agent_start and session_start behavior.

## Contexto

Esta task forma parte de O05 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Inject compact Rootline guidance before agent starts.

## Alcance

**In**:
1. Guidance tells the model when to use Rootline tools.
2. Guidance includes only high-value schema or command hints.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-detect-rootline-project-state.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Guidance tells the model when to use Rootline tools.
- Guidance includes only high-value schema or command hints.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Pi before_agent_start docs`
- `integrations/pi/extensions/`
