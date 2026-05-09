---
estado: Specified
tipo: task
---
# T006: Verify the package locally through Pi.

**Outcome**: [O03 Build read-only Pi package MVP](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T005-add-read-only-tool-tests.md]]

## Preserva

- INV1: MVP tools do not mutate repository files.
  - Verificar: Inspect extension tool implementations and tests.

## Contexto

Esta task forma parte de O03 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Verify the package locally through Pi.

## Alcance

**In**:
1. Local install or direct extension load succeeds.
2. A headless Pi prompt can use at least one Rootline tool successfully.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T005-add-read-only-tool-tests.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Local install or direct extension load succeeds.
- A headless Pi prompt can use at least one Rootline tool successfully.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/package.json`
- `.pi/settings.json documentation`
