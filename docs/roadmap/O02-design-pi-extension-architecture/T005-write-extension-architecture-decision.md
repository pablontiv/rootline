---
estado: In Progress
tipo: task
---
# T005: Write the architecture decision record for the Pi Rootline extension.

**Outcome**: [O02 Design Pi extension architecture](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T004-design-permission-model.md]]

## Preserva

- INV1: The extension treats Rootline CLI JSON as the integration boundary; it does not import Go internal packages.
  - Verificar: Check architecture docs and implementation tasks.

## Contexto

Esta task forma parte de O02 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Write the architecture decision record for the Pi Rootline extension.

## Alcance

**In**:
1. ADR covers package layout, tool naming, CLI boundary, permissions, and compatibility.
2. ADR becomes source of truth for implementation Outcomes.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T004-design-permission-model.md` está completada o su salida está disponible.

## Criterios de Aceptación

- ADR covers package layout, tool naming, CLI boundary, permissions, and compatibility.
- ADR becomes source of truth for implementation Outcomes.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `docs/roadmap/O01-map-rootline-integration-surface/`
- `docs/roadmap/O02-design-pi-extension-architecture/`
