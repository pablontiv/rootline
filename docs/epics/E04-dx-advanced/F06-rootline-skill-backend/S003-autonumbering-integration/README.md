---
estado: Specified
tipo: historia
cliente: Platform Owner
---
# S003: Auto-numbering Integration

**Feature**: [F06 Rootline Skill Backend](../README.md)
**Capacidad**: El skill /roadmap usa `rootline describe --field schema.id.next` para auto-numbering y `rootline new` para scaffolding, eliminando los bash one-liners de ls|sort|tail

## Antes / Despues

**Antes**: El task-guide.md calcula el proximo numero de task con `ls T[0-9][0-9][0-9]-*.md 2>/dev/null | sort -V | tail -1` — bash puro que falla silenciosamente y no usa el schema. El frontmatter se escribe manualmente, pudiendo incluir valores invalidos de enum.

**Despues**: `rootline describe <story-dir> --field schema.id.next` retorna el proximo ID (ej: "T004") directamente. `rootline new <path>` genera el frontmatter con los valores de enum correctos segun el .stem. El skill es rootline-first en todas las operaciones.

## Criterios de Aceptacion (semanticos)

- [ ] task-guide.md Paso 3 usa `rootline describe --field schema.id.next`, no `ls|sort`
- [ ] task-guide.md Paso 4 incluye `rootline new <path>` antes de editar el archivo
- [ ] SKILL.md tiene seccion "Comandos Rootline de Referencia" con tabla de 7 comandos
- [ ] `/roadmap view` incluye `rootline stats` ademas de `rootline tree`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-replace-autonumbering-bash.md) | Reemplazar ls\|sort en task-guide con rootline describe |
| [T002](T002-add-rootline-new-scaffolding.md) | Agregar rootline new al paso de scaffolding |
| [T003](T003-add-stats-to-view.md) | Agregar rootline stats a /roadmap view |

## Dependencias

- F07/S001 debe completarse antes de T001 y T002 (requiere `rootline describe --field schema.id.next` funcionando)

## Fuente de verdad

- `.claude/skills/roadmap/task-guide.md` — Paso 3 y Paso 4 a modificar
- `.claude/skills/roadmap/SKILL.md` — seccion view y referencia a agregar
- `internal/rules/describe.go` — implementacion de sequence (F07)
