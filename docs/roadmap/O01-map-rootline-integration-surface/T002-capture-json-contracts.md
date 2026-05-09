---
estado: Specified
tipo: task
---
# T002: Capture JSON output contracts for commands that Pi can consume.

**Outcome**: [O01 Map Rootline integration surface](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T001-inventory-rootline-commands.md]]

## Preserva

- INV1: No Pi tool is implemented before its Rootline command contract and risk class are documented.
  - Verificar: Check downstream tasks reference this Outcome as source of truth.

## Contexto

Esta task forma parte de O01 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Capture JSON output contracts for commands that Pi can consume.

## Alcance

**In**:
1. For each candidate command, document version/kind shape or note missing contract.
2. Identify commands whose output should not be parsed until normalized.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-inventory-rootline-commands.md` está completada o su salida está disponible.

## Criterios de Aceptación

- For each candidate command, document version/kind shape or note missing contract.
- Identify commands whose output should not be parsed until normalized.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `cmd/rootline/*.go`
- `internal/* result/report types`
