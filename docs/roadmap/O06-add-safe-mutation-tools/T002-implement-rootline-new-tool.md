---
estado: In Progress
tipo: task
---
# T002: Implement rootline_new for creating governed records.

**Outcome**: [O06 Add safe mutation tools](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T001-design-mutation-guardrails.md]]

## Preserva

- INV1: Mutating tools must not bypass Rootline validation or write outside user-approved paths.
  - Verificar: Review tool implementation and tests.

## Contexto

Esta task forma parte de O06 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Implement rootline_new for creating governed records.

## Alcance

**In**:
1. Tool calls rootline new for approved paths.
2. Tool validates the created file and returns structured result details.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-design-mutation-guardrails.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Tool calls rootline new for approved paths.
- Tool validates the created file and returns structured result details.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `cmd/rootline/new.go`
- `integrations/pi/extensions/`
