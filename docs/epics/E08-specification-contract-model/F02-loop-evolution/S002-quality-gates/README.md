---
estado: Pending
tipo: historia
---
# S002: Quality Gates

**Feature**: [F02 Loop Evolution](../README.md)
**Capacidad**: El loop de ejecucion ejecuta /security-review y /review como quality gates automaticos
**Cubre**: Milestone de F02 — "/roadmap loop ejecuta /security-review selectivo y /review por checkpoint"

## Antes / Despues

**Antes**: La unica validacion post-implementacion son los ACs binarios (paso 6). No hay review de calidad de codigo ni analisis de seguridad. El loop puede ejecutar decenas de tasks sin feedback de calidad.

**Despues**: /security-review se ejecuta selectivamente en tasks con superficie de ataque (pre-push). /review se ejecuta por checkpoint (cambio de Story, cada N tasks, loop interrumpido). Metricas de review aparecen en el resumen final. Flags --checkpoint-interval y --skip-reviews controlan el comportamiento.

## Invariantes

- INV1: El loop existente sigue funcionando sin --skip-reviews
- INV2: Quality gates no bloquean el loop excepto /security-review HIGH findings

## Criterios de Aceptacion (semanticos)

- [ ] SKILL.md loop tiene paso de security review selectivo (post-ACs, pre-commit)
- [ ] SKILL.md loop tiene checkpoint detection con /review
- [ ] Resumen final incluye metricas de reviews
- [ ] Flags --checkpoint-interval y --skip-reviews documentados

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-security-review-step.md) | Agregar security review selectivo al loop |
| [T002](T002-add-checkpoint-review.md) | Agregar checkpoint review con /review |
| [T003](T003-update-loop-summary.md) | Actualizar resumen final + verificar invariantes |

## Fuente de verdad

- `.claude/skills/roadmap/SKILL.md`
