---
estado: Obsolete
tipo: task
---
# T006: Design or implement rootline pi init onboarding helper.

**Outcome**: [O08 Productionize testing, release, and adoption](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T005-write-user-adoption-docs.md]]

## Preserva

- INV1: Release automation must not publish unvalidated extension code or include unintended files.
  - Verificar: Review package files, CI, and npm/git release config.

## Contexto

Esta task forma parte de O08 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Design or implement rootline pi init onboarding helper.

## Alcance

**In**:
1. Decision states whether rootline pi init is included now or deferred.
2. If implemented, it scaffolds Pi settings without hiding what will be installed.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T005-write-user-adoption-docs.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Decision states whether rootline pi init is included now or deferred.
- If implemented, it scaffolds Pi settings without hiding what will be installed.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `cmd/rootline/`
- `install.sh`
- `integrations/pi/`
