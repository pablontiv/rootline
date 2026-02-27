---
estado: Completed
tipo: feature
---
# F01: Contract-Based Specification

**Epic**: [E08](../README.md)
**Objetivo**: El framework de planificacion define contratos formales (Pre/Post/Invariantes/Traza) en cada nivel de la jerarquia
**Beneficio**: Tasks trazan bidireccionalmente a requisitos superiores y Stories tienen invariantes medibles que el loop verifica
**Milestone**: framework-reference.md tiene seccion 2.3 + contratos en 4.1-4.4; SKILL.md tiene Paso 2.5; todos los templates tienen campos de trazabilidad
**Satisface**: P1 (trazabilidad), P2 (invariantes medibles) — del [Epic](../README.md)

## Scope

**In**: Modelo de contratos en framework-reference.md, Paso 2.5 en SKILL.md, campos de trazabilidad en epic/story/task-guide.md, subagente sdd-validator
**Out**: Cambios al motor rootline, nuevos campos en .stem schemas, migracion de epics existentes

## Invariantes

- INV1: Los archivos del skill existentes siguen funcionando sin regression
- INV2: Todos los cambios son aditivos

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-framework-contract-definitions/) | Framework Contract Definitions | El framework define contratos formales por nivel |
| [S002](S002-workflow-integration/) | Workflow Integration | El workflow autonomo formaliza contratos antes de descomponer |
| [S003](S003-template-traceability-fields/) | Template Traceability Fields | Cada template tiene campos de trazabilidad y invariantes |

## Dependencias

- Ninguna — este es el foundation Feature

## Fuente de verdad

- `.claude/skills/roadmap/framework-reference.md`
- `.claude/skills/roadmap/SKILL.md`
- `.claude/skills/roadmap/epic-guide.md`
- `.claude/skills/roadmap/story-guide.md`
- `.claude/skills/roadmap/task-guide.md`
