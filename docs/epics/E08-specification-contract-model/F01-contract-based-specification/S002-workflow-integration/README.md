---
estado: Pending
tipo: historia
---
# S002: Workflow Integration

**Feature**: [F01 Contract-Based Specification](../README.md)
**Capacidad**: El workflow autonomo del skill formaliza contratos antes de descomponer y valida propagacion
**Cubre**: Milestone de F01 — "SKILL.md tiene Paso 2.5"

## Antes / Despues

**Antes**: El modo autonomo salta de absorber contexto (Paso 2) a aplicar framework (Paso 3) sin formalizar restricciones. La validacion de completitud (Paso 4.5) usa checks informales. No hay validacion automatica de trazabilidad post-materialization.

**Despues**: Paso 2.5 formaliza postcondiciones, invariantes y out of scope antes de descomponer. Pasos 3/4/4.5 usan contratos formales. Un subagente valida la cadena de trazabilidad automaticamente.

## Invariantes

- INV1: El contenido existente de SKILL.md no se elimina ni modifica (solo adiciones)
- INV2: Los subcomandos existentes (pending, loop, plan) siguen funcionando

## Criterios de Aceptacion (semanticos)

- [ ] SKILL.md tiene Paso 2.5 "Formalizar Contratos" entre Paso 2 y Paso 3
- [ ] Paso 4.5 tiene checks basados en contratos (no informales)
- [ ] Existe subagente sdd-validator en .claude/agents/

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-paso-2-5-to-skill.md) | Agregar Paso 2.5 a SKILL.md |
| [T002](T002-reinforce-validation-steps.md) | Reforzar Pasos 3/4/4.5 con contratos formales |
| [T003](T003-create-sdd-validator-agent.md) | Crear subagente sdd-validator |

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md`
- `.claude/agents/sdd-validator.md`
