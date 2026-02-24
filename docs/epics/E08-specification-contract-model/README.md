---
estado: In Progress
tipo: feature
---
# E08: Specification Contract Model

**Metrica de exito**: Toda task generada por /roadmap traza bidireccionalmente a un requisito superior, tiene invariantes medibles, y pasa quality gates automaticos
**Timeline**: 2026-Q1 — en curso

## Intencion

El skill /roadmap genera tasks que tienen criterios de aceptacion binarios pero desconectados del objetivo superior. No hay mecanismo formal para garantizar que una task "cumple requisitos" ni para medir propiedades del sistema que deben preservarse entre tasks. Este Epic introduce un modelo de contratos (Pre/Post/Invariantes/Traza) basado en Design by Contract, integra tipos dinamicos via rootline, y agrega quality gates (/security-review, /review) al loop de ejecucion.

## Postcondiciones

| ID | Condicion | Features |
|----|-----------|----------|
| P1 | Cada task traza a un criterio de story via "Contribuye a" | F01 |
| P2 | Stories tienen invariantes con comandos de verificacion ejecutables | F01 |
| P3 | /roadmap loop ejecuta /security-review y /review como quality gates | F02 |
| P4 | Tipos de task se descubren via rootline describe, no hardcodeados | F02 |

## Invariantes

- INV1: Los archivos del skill existentes siguen funcionando sin regression
- INV2: Todos los cambios son aditivos (no rompen workflows existentes)

## Out of Scope

- Regeneracion automatica de artefactos cuando cambia la spec
- RLM puro (context isolation via REPL programatico)
- Constitutional articles formales con proceso de enmienda
- Cambios al motor rootline (Go code)

## Features

| ID | Nombre | Descripcion |
|----|--------|-------------|
| [F01](F01-contract-based-specification/) | Contract-Based Specification | Modelo Pre/Post/Invariantes/Traza en framework, workflow y templates |
| [F02](F02-loop-evolution/) | Loop Evolution | Tipos dinamicos via rootline + quality gates en el loop |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | Foundation: define el modelo de contratos |
| F02 | F01 | Loop evolution usa invariantes definidos en F01 |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-02-24 | Modelo Pre/Post/Invariantes/Traza en vez de 5 capas SDD | Solo Spec + Validation layers son relevantes al objetivo |
| 2026-02-24 | Design by Contract como base en vez de SDD puro | DbC aporta invariantes — la pieza faltante para "medible" |
| 2026-02-24 | Tipos extraidos a archivo separado via rootline | Portabilidad entre proyectos sin editar skill core |
| 2026-02-24 | /security-review selectivo + /review por checkpoint | Balance costo vs feedback; no review en cada task |

## Gaps Activos

- Pendiente definir si invariantes se agregan al .stem schema o son solo markdown
