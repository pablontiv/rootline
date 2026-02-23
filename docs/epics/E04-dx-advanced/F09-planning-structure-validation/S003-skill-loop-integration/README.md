---
tipo: historia
cliente: Platform Owner
---
# S003: Skill and Loop Integration

**Feature**: [F09 Planning Structure Validation](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Los skills de roadmap generan wiki-links de dependencia y /roadmap loop usa rootline graph para respetar el orden de ejecucion

## Antes / Despues

**Antes**: El template de task-guide.md no documenta como declarar dependencias con wiki-links. `/roadmap loop` ejecuta tasks en orden de discovery (filesystem) sin considerar dependencias. El .stem de docs/epics/ no tiene reglas estructurales configuradas. Los epic directories E03 y E04 no tienen README.md.

**Despues**: task-guide.md documenta uso de `[[blocks:TXXX-name]]` para dependencias. `/roadmap loop` usa `rootline graph --check` para validar dependencias y `rootline graph` para resolver orden de ejecucion. `docs/epics/.stem` tiene `structural:` configurado. E03 y E04 tienen README.md.

## Criterios de Aceptacion (semanticos)

- [ ] `/roadmap task` genera `[[blocks:X]]` en el body cuando hay dependencias
- [ ] `/roadmap loop` salta tasks cuyas dependencias no estan Completado, con mensaje explicativo
- [ ] `rootline validate --all docs/epics/` ejecuta sin warnings estructurales despues de configurar

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-update-task-guide-with-blocks.md) | Actualizar template de task-guide.md para usar wiki-links de dependencia |
| [T002](T002-update-loop-dependency-order.md) | Actualizar /roadmap loop para usar rootline graph en orden de ejecucion |
| [T003](T003-configure-epics-stem-structural.md) | Configurar structural rules en docs/epics/.stem y crear READMEs faltantes |

## Fuente de verdad

- `.claude/skills/roadmap/task-guide.md` — template de tasks
- `.claude/skills/roadmap/SKILL.md` — /roadmap loop procedure
- `docs/epics/.stem` — schema a configurar
