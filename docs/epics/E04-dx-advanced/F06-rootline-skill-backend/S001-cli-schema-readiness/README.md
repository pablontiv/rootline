---
estado: Specified
tipo: historia
cliente: Platform Owner
---
# S001: CLI & Schema Readiness

**Feature**: [F06 Rootline Skill Backend](../README.md)
**Capacidad**: rootline query `in` operator funciona con valores comma-separated y el .stem tipo enum cubre los 14 tipos de task del framework

## Antes / Despues

**Antes**: `rootline query --where "estado in Pending,Especificado"` falla porque Cobra `StringSliceVar` splitea por coma antes de que el parser vea la expresion completa. El enum `tipo` en `docs/epics/.stem` solo tiene 4 valores (software-module, ci-cd, feature, historia) cuando el task-guide define 14 tipos.

**Despues**: El operador `in` funciona con valores comma-separated. El .stem cubre todos los tipos de task (IaC, software, general), permitiendo que `rootline validate` y `rootline new` manejen cualquier tipo correctamente.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline query --where "estado in Pending,Especificado"` retorna records sin error
- [ ] `rootline validate --all docs/epics/` pasa con schema ampliado
- [ ] `rootline new --dry-run` muestra 14 tipos en el comment del campo tipo

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-fix-query-in-operator.md) | Cambiar StringSliceVar a StringArrayVar en query.go |
| [T002](T002-expand-stem-tipo-enum.md) | Agregar 10 valores faltantes al enum tipo en .stem |

## Fuente de verdad

- `cmd/rootline/query.go:31` — StringSliceVar flag definition
- `docs/epics/.stem` — schema actual con 4 tipos
- `.claude/skills/roadmap/task-guide.md` — definicion de los 14 tipos
