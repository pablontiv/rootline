---
estado: Specified
tipo: task
---
# T001: Inventory every Rootline CLI command and its flags that matter to Pi.

**Outcome**: [O01 Map Rootline integration surface](README.md)
**Contribuye a**: CE1 del Outcome.

## Preserva

- INV1: No Pi tool is implemented before its Rootline command contract and risk class are documented.
  - Verificar: Check downstream tasks reference this Outcome as source of truth.

## Contexto

Esta task forma parte de O01 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Inventory every Rootline CLI command and its flags that matter to Pi.

## Alcance

**In**:
1. Run ./rootline --help or rootline --help and capture the command list.
2. Record each command in a command inventory document inside this task.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.

## Criterios de Aceptación

- Run ./rootline --help or rootline --help and capture the command list.
- Record each command in a command inventory document inside this task.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Rootline CLI commands in cmd/rootline/`
- `README.md`
- `docs/*.md`
