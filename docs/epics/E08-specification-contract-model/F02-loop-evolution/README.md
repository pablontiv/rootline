---
estado: Pending
tipo: feature
---
# F02: Loop Evolution

**Epic**: [E08](../README.md)
**Objetivo**: El loop de ejecucion descubre tipos dinamicamente y ejecuta quality gates automaticos
**Beneficio**: El skill es portable entre proyectos (tipos no hardcodeados) y detecta problemas de seguridad/calidad durante ejecucion
**Milestone**: task-guide.md usa rootline describe para tipos; /roadmap loop ejecuta /security-review selectivo y /review por checkpoint
**Satisface**: P3 (quality gates), P4 (tipos dinamicos) — del [Epic](../README.md)

## Scope

**In**: Extraccion de tipos a type-specs.md, rootline describe como fuente de tipos, /security-review selectivo, /review por checkpoint, flags y metricas
**Out**: Nuevos tipos de task, cambios al motor rootline, hooks de Claude Code

## Invariantes

- INV1: Los archivos del skill existentes siguen funcionando sin regression
- INV2: Todos los cambios son aditivos

## Constraints hacia Stories

| Constraint | Story que la cubre |
|------------|--------------------|
| Tipos dinamicos via rootline | S001 |
| Quality gates en loop | S002 |

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-dynamic-type-discovery/) | Dynamic Type Discovery | Tipos de task se descubren via rootline, no hardcodeados |
| [S002](S002-quality-gates/) | Quality Gates | El loop ejecuta /security-review y /review automaticamente |

## Dependencias

- [F01](../F01-contract-based-specification/) — invariantes deben existir antes de que el loop los verifique

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md`
- `.claude/skills/roadmap/task-guide.md`
