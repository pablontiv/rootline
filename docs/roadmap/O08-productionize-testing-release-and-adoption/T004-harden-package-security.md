---
estado: Obsolete
tipo: task
---
# T004: Review package security and supply-chain posture.

**Outcome**: [O08 Productionize testing, release, and adoption](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T003-define-package-release-strategy.md]]

## Preserva

- INV1: Release automation must not publish unvalidated extension code or include unintended files.
  - Verificar: Review package files, CI, and npm/git release config.

## Contexto

Esta task forma parte de O08 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Review package security and supply-chain posture.

## Alcance

**In**:
1. Package has minimal runtime dependencies.
2. Published files are allowlisted and secrets are not exposed.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T003-define-package-release-strategy.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Package has minimal runtime dependencies.
- Published files are allowlisted and secrets are not exposed.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/package.json`
- `.npmignore or files allowlist`
