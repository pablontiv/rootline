---
estado: In Progress
tipo: task
---
# T003: Implement /rootline doctor command.

**Outcome**: [O04 Package skill, prompts, and command UX](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:../O03-build-read-only-pi-package-mvp/T002-implement-rootline-cli-runner.md]]

## Preserva

- INV1: Skills and prompts instruct agents to prefer Rootline tools over manual grep/read for governed records.
  - Verificar: Inspect packaged skill and prompt content.

## Contexto

Esta task forma parte de O04 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Implement /rootline doctor command.

## Alcance

**In**:
1. Doctor checks rootline binary, version, cwd, .stem presence, and representative JSON command.
2. Doctor reports actionable diagnostics without mutating files.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `../O03-build-read-only-pi-package-mvp/T002-implement-rootline-cli-runner.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Doctor checks rootline binary, version, cwd, .stem presence, and representative JSON command.
- Doctor reports actionable diagnostics without mutating files.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/extensions/`
