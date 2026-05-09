---
estado: Specified
tipo: task
---
# T003: Classify each command as Pi tool, slash command, prompt, context rule, or unsupported.

**Outcome**: [O01 Map Rootline integration surface](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T002-capture-json-contracts.md]]

## Preserva

- INV1: No Pi tool is implemented before its Rootline command contract and risk class are documented.
  - Verificar: Check downstream tasks reference this Outcome as source of truth.

## Contexto

Esta task forma parte de O01 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Classify each command as Pi tool, slash command, prompt, context rule, or unsupported.

## Alcance

**In**:
1. Every command from T001 appears exactly once in the classification matrix.
2. Mutating commands include an explicit risk class.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T002-capture-json-contracts.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Every command from T001 appears exactly once in the classification matrix.
- Mutating commands include an explicit risk class.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `T001-inventory-rootline-commands.md`
- `T002-capture-json-contracts.md`
