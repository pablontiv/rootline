---
estado: Pending
tipo: historia
---
# S003: Template Traceability Fields

**Feature**: [F01 Contract-Based Specification](../README.md)
**Capacidad**: Cada template de planificacion tiene campos de trazabilidad y invariantes
**Cubre**: Milestone de F01 — "todos los templates tienen campos de trazabilidad"

## Antes / Despues

**Antes**: Epic/Feature templates no tienen postcondiciones ni invariantes. Story template tiene Antes/Despues pero no declara que parte del Feature cubre. Task template tiene ACs pero no declara a que criterio de Story contribuye ni que invariantes debe preservar.

**Despues**: Epic template tiene Postcondiciones + Invariantes + Out of Scope. Feature template declara Satisface (→ Epic). Story template declara Cubre (→ Feature) + Invariantes. Task template declara Contribuye a (→ Story) + Preserva (→ invariantes). Checklist de Task tiene 7ma condicion de trazabilidad.

## Invariantes

- INV1: Los templates existentes no pierden campos actuales
- INV2: `rootline validate` sigue pasando en docs/epics/ existentes

## Criterios de Aceptacion (semanticos)

- [ ] epic-guide.md tiene Postcondiciones/Invariantes/OutOfScope en Epic template y Satisface/Invariantes en Feature template
- [ ] story-guide.md tiene Cubre + Invariantes con comandos de verificacion
- [ ] task-guide.md tiene Contribuye_a + Preserva + 7ma condicion en checklist

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-postcondiciones-to-epic-guide.md) | Agregar Postcondiciones/Invariantes a epic-guide.md |
| [T002](T002-add-cubre-to-story-guide.md) | Agregar Cubre/Invariantes a story-guide.md |
| [T003](T003-add-contribuye-a-to-task-guide.md) | Agregar Contribuye_a/Preserva a task-guide.md |

## Fuente de verdad

- `.claude/skills/roadmap/epic-guide.md`
- `.claude/skills/roadmap/story-guide.md`
- `.claude/skills/roadmap/task-guide.md`
