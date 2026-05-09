---
estado: Completed
tipo: task
---
# T004: Prototype autocomplete for schema fields and record references.

**Outcome**: [O05 Add Rootline-aware runtime context](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T002-inject-rootline-agent-guidance.md]]

## Preserva

- INV1: Injected context must be compact and derived from current repo state, not stale cached assumptions.
  - Verificar: Inspect before_agent_start and session_start behavior.

## Contexto

Esta task forma parte de O05 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Prototype autocomplete for schema fields and record references.

## Alcance

**In**:
1. Autocomplete suggestions are sourced from rootline describe/query.
2. Prototype can be disabled if noisy or slow.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T002-inject-rootline-agent-guidance.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Autocomplete suggestions are sourced from rootline describe/query.
- Prototype can be disabled if noisy or slow.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Pi autocomplete docs`
- `rootline describe`
