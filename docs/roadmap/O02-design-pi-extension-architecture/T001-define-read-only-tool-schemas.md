---
estado: Specified
tipo: task
---
# T001: Define parameter and result contracts for read-only Rootline tools.

**Outcome**: [O02 Design Pi extension architecture](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:../O01-map-rootline-integration-surface/T003-classify-pi-exposure.md]]

## Preserva

- INV1: The extension treats Rootline CLI JSON as the integration boundary; it does not import Go internal packages.
  - Verificar: Check architecture docs and implementation tasks.

## Contexto

Esta task forma parte de O02 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Define parameter and result contracts for read-only Rootline tools.

## Alcance

**In**:
1. Schemas exist for query, tree, validate, describe, stats, graph/explain if selected.
2. Each schema avoids arbitrary shell command strings.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `../O01-map-rootline-integration-surface/T003-classify-pi-exposure.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Schemas exist for query, tree, validate, describe, stats, graph/explain if selected.
- Each schema avoids arbitrary shell command strings.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `docs/roadmap/O01-map-rootline-integration-surface/`
