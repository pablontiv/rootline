---
estado: Pending
tipo: feature
---
# F06: Rootline Skill Backend

**Epic**: [E04](../README.md)
**Objetivo**: El skill /roadmap usa rootline CLI como engine para scaffolding, validacion y consultas, eliminando logica duplicada
**Beneficio**: Elimina 100+ lineas de logica hardcodeada (Python script, manual tree, templates), asegura que documentos creados pasan validacion rootline, y corrige el scope post-aprobacion (solo crear archivos, no implementar)
**Milestone**: `/roadmap view` ejecuta `rootline tree`, `/roadmap pending` ejecuta `rootline query`, documentos creados pasan `rootline validate`

## Scope

**In**: Fix query `in` operator, ampliar .stem enum tipo, reescribir SKILL.md pending/view/post-approval con rootline commands, agregar validate post-write, habilitar model invocation
**Out**: Cambios al framework-reference.md o task-guide.md, nuevos comandos rootline, cambios a /roadmap loop

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [CLI & Schema Readiness](S001-cli-schema-readiness/) | Query `in` funciona y .stem cubre todos los tipos de task |
| S002 | [Skill Rootline Integration](S002-skill-rootline-integration/) | Skill usa rootline query/tree/validate en vez de logica hardcodeada |
| S003 | [Auto-numbering Integration](S003-autonumbering-integration/) | Skill usa rootline describe para IDs y rootline new para scaffolding |

## Dependencias

- Ninguna Feature previa requerida
- S001 debe completarse antes de S002
- F07/S001 debe completarse antes de S003

## Fuente de verdad

- `cmd/rootline/query.go` — query command (StringSliceVar bug)
- `docs/epics/.stem` — schema para documentos de epics
- `.claude/skills/roadmap/SKILL.md` — skill principal a reescribir
