---
estado: In Progress
tipo: task
---
# T002: Design a shared CLI runner for executing rootline from Pi.

**Outcome**: [O02 Design Pi extension architecture](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T001-define-read-only-tool-schemas.md]]

## Preserva

- INV1: The extension treats Rootline CLI JSON as the integration boundary; it does not import Go internal packages.
  - Verificar: Check architecture docs and implementation tasks.

## Contexto

Esta task forma parte de O02 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Design a shared CLI runner for executing rootline from Pi.

## Alcance

**In**:
1. Runner design covers cwd, binary path, timeouts, abort signals, JSON parsing, and stderr.
2. Runner design defines stable error objects for the model.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-define-read-only-tool-schemas.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Runner design covers cwd, binary path, timeouts, abort signals, JSON parsing, and stderr.
- Runner design defines stable error objects for the model.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Pi extension docs`
- `Rootline CLI behavior`
