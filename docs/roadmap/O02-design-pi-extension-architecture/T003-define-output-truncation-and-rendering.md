---
estado: In Progress
tipo: task
---
# T003: Define output truncation and optional TUI rendering behavior.

**Outcome**: [O02 Design Pi extension architecture](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T002-design-rootline-cli-runner.md]]

## Preserva

- INV1: The extension treats Rootline CLI JSON as the integration boundary; it does not import Go internal packages.
  - Verificar: Check architecture docs and implementation tasks.

## Contexto

Esta task forma parte de O02 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Define output truncation and optional TUI rendering behavior.

## Alcance

**In**:
1. Large outputs have documented truncation limits and full-output handoff behavior.
2. Tool results distinguish model-facing content from details.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T002-design-rootline-cli-runner.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Large outputs have documented truncation limits and full-output handoff behavior.
- Tool results distinguish model-facing content from details.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Pi extension output truncation docs`
