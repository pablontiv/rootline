---
estado: Completed
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T002: Agregar idempotencia y diff-guard al workflow

**Story**: [S002 Cross-Repo Sync Pipeline](README.md)

## Contexto

El usuario pushea muchas veces al dia. No todos los pushes cambian skills — muchos son cambios de código Go o docs. El workflow debe ser no-op cuando `.claude/skills/` no tiene cambios reales respecto al marketplace, evitando commits vacíos y ruido en el historial.

## Alcance

**In**:
1. Comparación de hash de contenido entre skills fuente y destino
2. Exit temprano si no hay diferencias después del rsync
3. Trigger `workflow_dispatch` para re-sync manual forzado
4. Log claro indicando "no changes detected, skipping push"

**Out**: Validación de estructura (T003), nuevos triggers

## Estado inicial esperado

- T001 completado: workflow base funcional

## Criterios de Aceptacion

- Push sin cambios en skills no genera commit en agent-marketplace
- Push con cambios en skills genera commit normalmente
- `workflow_dispatch` fuerza sync incluso sin cambios
- Logs del workflow indican claramente si hubo sync o skip

## Fuente de verdad

- `.github/workflows/publish-marketplace.yml` (workflow a modificar)
