---
estado: Completed
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

## Implementación

Creado `integrations/pi/extensions/status.ts` que registra la herramienta `rootline-status`:

- **Función**: Health check rápida (<3s típico) para widgets de UI
- **Estados detectados**: no_rootline (binary not found), binary_only (no .stem), stem_governed (governed + validation)
- **Salida**: JSON con state, status_line (human-readable), version, stem_count, valid, errors, warnings
- **Símbolos**: ✗ (no_rootline), ⚠ (binary_only), ✓ (stem_governed)
- **Tests**: 6 tests unitarios verifican registro, esquema y contrato de comportamiento

**Criterios de aceptación verificados**:
1. Status shows missing binary, no .stem, valid, or errors ✓
2. Widget output remains concise and non-blocking (5s timeout) ✓
3. `rootline validate --all docs/roadmap/` exit 0 ✓ (79 valid, 0 errors, 2 warnings)

**Commit**: feat(pi): add rootline-status widget for project health monitoring
