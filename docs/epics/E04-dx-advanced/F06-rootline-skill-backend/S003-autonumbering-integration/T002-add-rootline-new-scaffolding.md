---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T002: Agregar rootline new al paso de scaffolding

**Story**: [S003 Auto-numbering Integration](README.md)

## Contexto

El task-guide.md Paso 4 dice "Crear el archivo .md" y lo hace directamente via Write con frontmatter hardcodeado. Esto puede producir valores invalidos de enum (el agente escribe el tipo que le parece correcto sin consultar el schema). `rootline new <path>` genera el frontmatter segun el .stem del directorio, con los valores de enum correctos y comentados.

## Dependencias

- F07/S001 completado (rootline new debe conocer `type: sequence` sin errores)

## Alcance

**In**:
1. En task-guide.md Paso 4, agregar `rootline new <story-dir>/TXXX-nombre.md` como primer sub-paso antes de editar el contenido
2. Agregar nota: "El frontmatter generado por rootline new es el punto de partida; editar el contenido del task pero no el schema del frontmatter"
3. Mismo patron en story-guide.md para creacion de Stories

**Out**: Cambios al comportamiento de `rootline new`, cambios a otros pasos del guide

## Estado inicial esperado

- `rootline new` funciona y genera frontmatter segun el .stem del directorio target
- .stem en nivel Story incluye campos estado, tipo, ejecutable_en con enums correctos

## Criterios de Aceptacion

- task-guide.md Paso 4 incluye `rootline new <path>` como sub-paso 4.1
- La instruccion dice explicitamente que el agente edita el contenido, no el frontmatter
- Ejecutar `rootline new /tmp/test-task.md` en directorio con .stem de Story genera frontmatter valido
- `rootline validate /tmp/test-task.md` pasa sin errores despues de solo ejecutar `rootline new`

## Fuente de verdad

- `.claude/skills/roadmap/task-guide.md` — Paso 4: Generar Task File
- `.claude/skills/roadmap/story-guide.md` — Paso 4: Generar Story
- `cmd/rootline/new.go` — implementacion del comando new
