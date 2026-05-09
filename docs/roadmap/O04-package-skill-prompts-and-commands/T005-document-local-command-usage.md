---
estado: Completed
tipo: task
---
# T005: Document local usage of tools, prompts, and commands.

**Outcome**: [O04 Package skill, prompts, and command UX](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T004-implement-validation-and-tree-commands.md]]

## Preserva

- INV1: Tool descriptions and prompts instruct agents to prefer Rootline tools over manual grep/read for governed records.
  - Verificar: Inspect packaged tool descriptions and prompt content.

## Contexto

Esta task forma parte de O04 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Document local usage of tools, prompts, and commands.

## Alcance

**In**:
1. Docs show install, reload, tool examples, and command examples.
2. Docs include troubleshooting for missing rootline binary.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T004-implement-validation-and-tree-commands.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Docs show install, reload, tool examples, and command examples.
- Docs include troubleshooting for missing rootline binary.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `README.md`
- `integrations/pi/README.md`
