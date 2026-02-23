---
tipo: historia
cliente: Platform Owner
---
# S002: Skill Rootline Integration

**Feature**: [F06 Rootline Skill Backend](../README.md)
**Capacidad**: El skill /roadmap usa rootline query para pending, rootline tree para view, rootline validate post-write, y solo crea archivos .md post-aprobacion

## Antes / Despues

**Antes**: `/roadmap pending` ejecuta 60 lineas de Python embebido con path hardcodeado a `/opt/homeserver/automation`. `/roadmap view` tiene 40 lineas de instrucciones manuales para escanear y renderizar arbol ASCII. Post-aprobacion intenta implementar tasks directamente. No hay validacion de documentos creados. `disable-model-invocation: true` impide invocacion programatica.

**Despues**: `/roadmap pending` ejecuta `rootline query --output table`. `/roadmap view` ejecuta `rootline tree --output table`. Post-aprobacion solo crea archivos .md y valida cada uno con `rootline validate`. Todos los skills son model-invocable.

## Criterios de Aceptacion (semanticos)

- [ ] Seccion `/roadmap pending` del SKILL.md usa rootline query, no Python
- [ ] Seccion `/roadmap view` del SKILL.md usa rootline tree, no instrucciones manuales
- [ ] Fase 2 (post-aprobacion) explicitamente dice "solo crear archivos"
- [ ] Cada subcomando de creacion incluye paso `rootline validate`
- [ ] No existe `disable-model-invocation` en frontmatter del skill

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-replace-pending-with-query.md) | Reemplazar script Python por rootline query |
| [T002](T002-replace-view-with-tree.md) | Reemplazar instrucciones manuales por rootline tree |
| [T003](T003-rewrite-post-approval-scope.md) | Reescribir post-aprobacion: solo crear .md + validate + enable model invocation |

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md` — skill a reescribir
- `cmd/rootline/query.go` — query command syntax
- `cmd/rootline/tree.go` — tree command output
