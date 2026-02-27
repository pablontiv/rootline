---
estado: Completed
tipo: historia
---
# S001: Dynamic Type Discovery

**Feature**: [F02 Loop Evolution](../README.md)
**Capacidad**: Tipos de task se descubren dinamicamente via rootline en vez de estar hardcodeados en el skill
**Cubre**: Milestone de F02 — "task-guide.md usa rootline describe para tipos"

## Antes / Despues

**Antes**: task-guide.md tiene 12 tipos hardcodeados con templates YAML inline. Si un proyecto usa tipos diferentes (k8s-workload, terraform-module), hay que editar el skill. Los .stem files duplican la lista de tipos.

**Despues**: task-guide.md indica `rootline describe <story-dir> --field schema.tipo` para descubrir tipos validos. Templates YAML viven en type-specs.md separado. Cada proyecto puede tener sus propios tipos sin editar el skill core.

## Invariantes

- INV1: `rootline describe <story-dir> --field schema.tipo` retorna tipos validos correctamente
- INV2: Los task files existentes siguen validando

## Criterios de Aceptacion (semanticos)

- [ ] task-guide.md no tiene lista hardcodeada de tipos en el template
- [ ] task-guide.md indica usar rootline describe para descubrir tipos
- [ ] type-specs.md existe con templates YAML extraidos de task-guide.md

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-extract-type-specs.md) | Extraer templates de especificacion tecnica a type-specs.md |
| [T002](T002-replace-hardcoded-types.md) | Reemplazar tipos hardcodeados con rootline describe |

## Fuente de verdad

- `.claude/skills/roadmap/task-guide.md`
- `.claude/skills/roadmap/type-specs.md` (nuevo)
