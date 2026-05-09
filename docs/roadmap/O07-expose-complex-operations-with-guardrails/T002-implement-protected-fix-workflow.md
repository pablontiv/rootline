---
estado: In Progress
tipo: task
---
# T002: Implement protected workflow for rootline fix.

**Outcome**: [O07 Expose complex operations with guardrails](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T001-design-complex-operation-ux.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T008-implement-schema-apply-explicit.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T009-implement-repair-apply-data-only.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T010-make-fix-all-schema-safe-by-default.md]]

## Preserva

- INV1: Bulk operations require explicit user intent and cannot be triggered silently by agent context guidance.
  - Verificar: Inspect commands, prompts, and tool activation defaults.

## Contexto

Esta task forma parte de O07 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Implement protected workflow for rootline fix.

## Alcance

**In**:
1. Workflow requires explicit target and confirmation when mutating.
2. Workflow validates after fixes and reports proposals/results.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-design-complex-operation-ux.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Workflow requires explicit target and confirmation when mutating.
- Workflow validates after fixes and reports proposals/results.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `internal/fix/`
- `cmd/rootline/fix.go`
- `integrations/pi/extensions/`
