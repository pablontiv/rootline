---
estado: Completed
tipo: task
---
# T005: Document complex operation risk model and rollback procedure.

**Outcome**: [O07 Expose complex operations with guardrails](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T004-implement-analyze-apply-workflow.md]]

## Preserva

- INV1: Bulk operations require explicit user intent and cannot be triggered silently by agent context guidance.
  - Verificar: Inspect commands, prompts, and tool activation defaults.

## Contexto

Esta task forma parte de O07 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Document complex operation risk model and rollback procedure.

## Alcance

**In**:
1. Docs explain when complex operations are disabled or require confirmation.
2. Docs include git checkpoint/rollback recommendations.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T004-implement-analyze-apply-workflow.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Docs explain when complex operations are disabled or require confirmation.
- Docs include git checkpoint/rollback recommendations.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `integrations/pi/README.md`
