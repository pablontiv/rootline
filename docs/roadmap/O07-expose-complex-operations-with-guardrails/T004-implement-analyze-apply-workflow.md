---
estado: Specified
tipo: task
---
# T004: Implement protected workflow around analyze reports and apply.

**Outcome**: [O07 Expose complex operations with guardrails](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T001-design-complex-operation-ux.md]]

## Preserva

- INV1: Bulk operations require explicit user intent and cannot be triggered silently by agent context guidance.
  - Verificar: Inspect commands, prompts, and tool activation defaults.

## Contexto

Esta task forma parte de O07 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Implement protected workflow around analyze reports and apply.

## Alcance

**In**:
1. Analyze report can be generated and inspected from Pi.
2. Apply requires explicit user approval, target report, and post-apply validation.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-design-complex-operation-ux.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Analyze report can be generated and inspected from Pi.
- Apply requires explicit user approval, target report, and post-apply validation.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `internal/infer/`
- `cmd/rootline/analyze.go`
- `cmd/rootline/apply.go`
