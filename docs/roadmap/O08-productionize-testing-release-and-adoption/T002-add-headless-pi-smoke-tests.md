---
estado: In Progress
tipo: task
---
# T002: Add headless Pi smoke tests for package discovery and core workflows.

**Outcome**: [O08 Productionize testing, release, and adoption](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T001-add-extension-ci-validation.md]]

## Preserva

- INV1: Release automation must not publish unvalidated extension code or include unintended files.
  - Verificar: Review package files, CI, and npm/git release config.

## Contexto

Esta task forma parte de O08 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Add headless Pi smoke tests for package discovery and core workflows.

## Alcance

**In**:
1. Smoke tests verify tools/commands are discoverable.
2. At least one read-only Rootline query succeeds through Pi.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-add-extension-ci-validation.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Smoke tests verify tools/commands are discoverable.
- At least one read-only Rootline query succeeds through Pi.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Pi CLI`
- `integrations/pi/`
