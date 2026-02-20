---
estado: Pending
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Reemplazar ls|sort en task-guide con rootline describe

**Story**: [S003 Auto-numbering Integration](README.md)

## Contexto

El task-guide.md Paso 3 usa `ls docs/epics/.../T[0-9][0-9][0-9]-*.md 2>/dev/null | sort -V | tail -1` para detectar el proximo numero de task. Este approach falla silenciosamente si no hay archivos, no usa el schema, y es bash puro. La feature `type: sequence` (F07/S001) expone `rootline describe <dir> --field schema.id.next` que retorna directamente el valor correcto.

## Dependencias

- F07/S001/T001 y T002 deben estar completados (SchemaField.Next implementado y .stem configurados)

## Alcance

**In**:
1. En task-guide.md Paso 3, reemplazar el bloque bash de ls|sort|tail con instruccion de ejecutar `rootline describe <story-dir> --field schema.id.next`
2. Actualizar el comentario explicativo del paso para reflejar el nuevo approach
3. Mismo cambio para auto-numbering de Stories (en story-guide.md) y Features (epic-guide.md)

**Out**: Cambios al engine rootline, cambios a SKILL.md principal

## Estado inicial esperado

- F07/S001 completado: `rootline describe <dir> --field schema.id.next` retorna "T004"
- `.stem` files configurados con `id: {type: sequence, prefix: T, digits: 3}` en nivel Story

## Criterios de Aceptacion

- task-guide.md Paso 3 contiene `rootline describe` como comando de auto-numbering
- No existe `ls T[0-9]` ni `sort -V` ni `tail -1` en task-guide.md
- story-guide.md y epic-guide.md actualizados con el mismo patron
- Ejecutar `rootline describe docs/epics/E04-dx-advanced/F07-sequence-autonumbering/S001-core-engine/ --field schema.id.next` retorna "T004"

## Fuente de verdad

- `.claude/skills/roadmap/task-guide.md` — Paso 3: Auto-numbering
- `.claude/skills/roadmap/story-guide.md` — Paso 3: Auto-numbering
- `.claude/skills/roadmap/epic-guide.md` — Paso 2: Auto-numbering
