---
estado: Specified
tipo: task
---
# T005: Write user-facing adoption and troubleshooting docs.

**Outcome**: [O08 Productionize testing, release, and adoption](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T004-harden-package-security.md]]

## Preserva

- INV1: Release automation must not publish unvalidated extension code or include unintended files.
  - Verificar: Review package files, CI, and npm/git release config.

## Contexto

Esta task forma parte de O08 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Write user-facing adoption and troubleshooting docs.

## Alcance

**In**:
1. Docs cover install, update, config, commands, tools, and common failures.
2. Docs include examples for project-local team install.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T004-harden-package-security.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Docs cover install, update, config, commands, tools, and common failures.
- Docs include examples for project-local team install.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `README.md`
- `integrations/pi/README.md`
- `docs/`
