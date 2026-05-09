---
estado: Specified
tipo: task
---
# T004: Implement rootline_validate, rootline_tree, and rootline_stats tools.

**Outcome**: [O03 Build read-only Pi package MVP](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T002-implement-rootline-cli-runner.md]]

## Preserva

- INV1: MVP tools do not mutate repository files.
  - Verificar: Inspect extension tool implementations and tests.

## Contexto

Esta task forma parte de O03 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Implement rootline_validate, rootline_tree, and rootline_stats tools.

## Alcance

**In**:
1. Tools call rootline with --output json when supported.
2. Validation results preserve errors and warnings for the model.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T002-implement-rootline-cli-runner.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Tools call rootline with --output json when supported.
- Validation results preserve errors and warnings for the model.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `cmd/rootline/validate.go`
- `cmd/rootline/tree.go`
- `cmd/rootline/stats.go`
