---
estado: In Progress
tipo: task
---
# T004: Add tests for success, validation failure, and blocked path cases.

**Outcome**: [O06 Add safe mutation tools](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T003-implement-rootline-set-tool.md]]

## Preserva

- INV1: Mutating tools must not bypass Rootline validation or write outside user-approved paths.
  - Verificar: Review tool implementation and tests.

## Contexto

Esta task forma parte de O06 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Add tests for success, validation failure, and blocked path cases.

## Alcance

**In**:
1. Tests cover rootline_new and rootline_set.
2. Tests assert failed validation is surfaced to the model.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T003-implement-rootline-set-tool.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Tests cover rootline_new and rootline_set.
- Tests assert failed validation is surfaced to the model.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi tests`
