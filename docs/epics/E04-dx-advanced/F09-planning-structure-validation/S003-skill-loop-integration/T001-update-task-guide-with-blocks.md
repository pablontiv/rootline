---
estado: Completado
tipo: documentation
---
# T001: Actualizar template de task-guide.md para usar wiki-links de dependencia

**Story**: [S003 Skill and Loop Integration](README.md)

## Contexto

El template de task en `.claude/skills/roadmap/task-guide.md` define la estructura que `/roadmap task` genera para cada task nuevo. Hoy la seccion `## Dependencias` es texto libre. Con el modelo de wiki-links, las dependencias machine-readable se declaran como `[[blocks:TXXX-name]]` en el body del task (tipicamente en la seccion Contexto, debajo del titulo de la Story link).

## Dependencias

- S002 completada (wiki-links funcionales en rootline)

## Alcance

**In**:
1. Agregar documentacion en task-guide.md sobre como declarar dependencias con wiki-links
2. Agregar ejemplo: `[[blocks:T001-prerequisite-task]]` en seccion Contexto del template
3. Explicar que `[[blocks:X]]` es leido por `rootline graph` para deteccion de ciclos y orden de ejecucion
4. Mantener seccion `## Dependencias` para contexto humano adicional (no machine-readable)

**Out**: No modificar el frontmatter template (blocks ya no va en YAML). No modificar otros skill guides.

## Estado inicial esperado

- `.claude/skills/roadmap/task-guide.md` no menciona wiki-links para dependencias
- Seccion `## Dependencias` es solo texto libre

## Criterios de Aceptacion

- task-guide.md documenta uso de `[[blocks:TXXX-name]]` para dependencias
- El template de task incluye ejemplo de wiki-link de dependencia en seccion Contexto
- Hay nota explicando que rootline graph lee estos links automaticamente
- La seccion `## Dependencias` del template aclara que es para contexto humano complementario

## Fuente de verdad

- `.claude/skills/roadmap/task-guide.md` — template actual
