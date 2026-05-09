---
estado: Completed
tipo: task
---
# T005: Add tests or executable fixtures for read-only tools.

**Outcome**: [O03 Build read-only Pi package MVP](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T003-implement-query-describe-tools.md]]

## Preserva

- INV1: MVP tools do not mutate repository files.
  - Verificar: Inspect extension tool implementations and tests.

## Contexto

Esta task forma parte de O03 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Add tests or executable fixtures for read-only tools.

## Alcance

**In**:
1. Tests cover success and missing-rootline failure paths.
2. Tests can run in CI without mutating project files.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T003-implement-query-describe-tools.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Tests cover success and missing-rootline failure paths.
- Tests can run in CI without mutating project files.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/`
- `test fixtures`
