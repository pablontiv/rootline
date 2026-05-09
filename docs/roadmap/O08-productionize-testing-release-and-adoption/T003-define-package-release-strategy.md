---
estado: Specified
tipo: task
---
# T003: Define release strategy for integrations/pi versus separate package.

**Outcome**: [O08 Productionize testing, release, and adoption](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T002-add-headless-pi-smoke-tests.md]]

## Preserva

- INV1: Release automation must not publish unvalidated extension code or include unintended files.
  - Verificar: Review package files, CI, and npm/git release config.

## Contexto

Esta task forma parte de O08 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Define release strategy for integrations/pi versus separate package.

## Alcance

**In**:
1. Decision covers npm package name, git install path, versioning, and compatibility.
2. Decision states whether package remains in this repo or moves later.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T002-add-headless-pi-smoke-tests.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Decision covers npm package name, git install path, versioning, and compatibility.
- Decision states whether package remains in this repo or moves later.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `package.json`
- `.goreleaser.yml`
- `README.md`
