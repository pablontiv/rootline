---
estado: In Progress
tipo: task
---
# T005: Document safe mutation workflows and non-goals.

**Outcome**: [O06 Add safe mutation tools](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T004-add-mutation-tests.md]]

## Preserva

- INV1: Mutating tools must not bypass Rootline validation or write outside user-approved paths.
  - Verificar: Review tool implementation and tests.

## Contexto

Esta task forma parte de O06 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Document safe mutation workflows and non-goals.

## Alcance

**In**:
1. Docs explain when to use rootline_new/set instead of edit/write.
2. Docs explicitly defer bulk operations to the complex operations Outcome.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T004-add-mutation-tests.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Docs explain when to use rootline_new/set instead of edit/write.
- Docs explicitly defer bulk operations to the complex operations Outcome.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/README.md`
