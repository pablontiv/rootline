---
estado: Specified
tipo: task
---
# T001: Design UX for complex Rootline operations.

**Outcome**: [O07 Expose complex operations with guardrails](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:../O06-add-safe-mutation-tools/T005-document-safe-mutation-workflows.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T001-codify-command-responsibility-contracts.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T008-implement-schema-apply-explicit.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T009-implement-repair-apply-data-only.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T011-deprecate-legacy-apply-and-update-command-surfaces.md]]

## Preserva

- INV1: Bulk operations require explicit user intent and cannot be triggered silently by agent context guidance.
  - Verificar: Inspect commands, prompts, and tool activation defaults.

## Contexto

Esta task forma parte de O07 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Design UX for complex Rootline operations.

## Alcance

**In**:
1. Design covers analyze, fix, migrate, and replacement schema/repair apply surfaces, or legacy apply only if O09 keeps it.
2. Each operation has user-intent, preview, validation, and rollback notes.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `../O06-add-safe-mutation-tools/T005-document-safe-mutation-workflows.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Design covers analyze, fix, migrate, and replacement schema/repair apply surfaces, or legacy apply only if O09 keeps it.
- Each operation has user-intent, preview, validation, and rollback notes.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `cmd/rootline/analyze.go`
- `cmd/rootline/fix.go`
- `cmd/rootline/migrate.go`
- `cmd/rootline/apply.go`
