---
estado: In Progress
tipo: task
---
# T003: Implement rootline_query and rootline_describe tools.

**Outcome**: [O03 Build read-only Pi package MVP](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T002-implement-rootline-cli-runner.md]]

## Preserva

- INV1: MVP tools do not mutate repository files.
  - Verificar: Inspect extension tool implementations and tests.

## Contexto

Esta task forma parte de O03 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Implement rootline_query and rootline_describe tools.

## Alcance

**In**:
1. rootline_query supports path, where, limit, sort/count where applicable.
2. rootline_describe supports path and optional field extraction.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T002-implement-rootline-cli-runner.md` está completada o su salida está disponible.

## Criterios de Aceptación

- rootline_query supports path, where, limit, sort/count where applicable.
- rootline_describe supports path and optional field extraction.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `cmd/rootline/query.go`
- `cmd/rootline/describe.go`
- `integrations/pi/extensions/`
