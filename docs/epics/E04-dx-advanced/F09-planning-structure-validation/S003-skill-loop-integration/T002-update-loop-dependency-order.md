---
estado: Completed
tipo: documentation
---
# T002: Actualizar /roadmap loop para usar rootline graph en orden de ejecucion

**Story**: [S003 Skill and Loop Integration](README.md)

[[blocks:T001-update-task-guide-with-blocks]]

## Contexto

`/roadmap loop` en `.claude/skills/roadmap/SKILL.md` ejecuta tasks pendientes en orden de discovery (filesystem). No verifica dependencias. Con wiki-links `[[blocks:X]]` en los tasks y `rootline graph` funcionando, el loop puede usar el graph para:

1. Validar que no hay ciclos antes de empezar (`rootline graph --check`)
2. Resolver orden topologico de ejecucion (tasks sin dependencias primero)
3. Saltar tasks cuyas dependencias no estan en estado Completado

## Dependencias

- T001 completado (template documenta wiki-links)
- S002 completada (wiki-links funcionales)

## Alcance

**In**:
1. En SKILL.md seccion "Fase 1: Discovery", agregar paso:
   - `rootline graph --check docs/epics/` para validar antes de empezar
   - Si hay ciclos o broken links → reportar y parar
2. En "Fase 3: Loop de Ejecucion", agregar paso antes de implementar:
   - Usar `rootline graph <story-dir> --format mermaid --output json` para leer edges
   - Para cada task: verificar que todos los targets de sus edges `blocks` estan en estado Completado
   - Si alguno no esta Completado → skip task con mensaje "Blocked by: TXXX (estado: Pending)"
3. Agregar logica de reintento: tasks bloqueados se ponen al final de la cola

**Out**: No implementar topological sort algoritmico. Usar heuristica simple: ejecutar tasks sin deps primero, luego reintentar bloqueados.

## Estado inicial esperado

- SKILL.md seccion "Fase 3" ejecuta tasks secuencialmente sin verificar dependencias
- No hay referencia a `rootline graph` en el loop

## Criterios de Aceptacion

- SKILL.md documenta `rootline graph --check` en Fase 1 como validacion pre-loop
- SKILL.md documenta verificacion de dependencias antes de ejecutar cada task
- Hay instruccion de skip con mensaje cuando un task esta bloqueado
- Hay instruccion de reintentar tasks bloqueados al final de la cola
- El flujo es: graph --check → para cada task: verificar deps → si bloqueado skip → si libre ejecutar

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md` — seccion /roadmap loop
