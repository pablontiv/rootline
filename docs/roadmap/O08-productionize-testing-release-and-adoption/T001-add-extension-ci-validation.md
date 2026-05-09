---
estado: In Progress
tipo: task
---
# T001: Add CI validation for the Pi package.

**Outcome**: [O08 Productionize testing, release, and adoption](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:../O07-expose-complex-operations-with-guardrails/T005-document-complex-operation-risks.md]]

## Preserva

- INV1: Release automation must not publish unvalidated extension code or include unintended files.
  - Verificar: Review package files, CI, and npm/git release config.

## Contexto

Esta task forma parte de O08 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Add CI validation for the Pi package.

## Alcance

**In**:
1. CI installs package dependencies and runs extension tests.
2. CI uses fixtures that cover read-only and mutation tools.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `../O07-expose-complex-operations-with-guardrails/T005-document-complex-operation-risks.md` está completada o su salida está disponible.

## Criterios de Aceptación

- CI installs package dependencies and runs extension tests.
- CI uses fixtures that cover read-only and mutation tools.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `.github/workflows/`
- `integrations/pi/`
