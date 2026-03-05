---
estado: Specified
tipo: feature
---
# E13: Inference Engine

**Metrica de exito**: `rootline analyze <path>` produce JSON report cubriendo categorias 1-13; `rootline apply` ejecuta inferencias aprobadas
**Timeline**: 2026-Q1

## Intencion

Rootline tiene un schema inference engine parcial (`internal/infer/`) que detecta tipos, enums, sequences, y aggregates (categorias 1-4). Este Epic extiende el engine para cubrir las 13 categorias de inferencia identificadas en la investigacion (intake/inference-engine-architecture.md), añade goldmark como parser AST para body content, y expone los resultados via los comandos `rootline analyze` y `rootline apply`.

La arquitectura sigue el modelo de 2 capas: engine (Go) + agent (LLM). Este Epic cubre solo el engine — las porciones Go de cada categoria. Las porciones LLM (cat 9/11 semantic residue) quedan como stubs para un Epic separado de integracion agent.

## Postcondiciones

- P1: `rootline analyze <path>` produce JSON report con `version: 1` cubriendo categorias 1-13
- P2: `rootline apply report.json` aplica cambios de schema y datos aprobados
- P3: `rootline analyze --incremental` detecta delta entre .stem actual y datos

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
- INV2: Contratos JSON mantienen `"version": 1` — additive-only
- INV3: Coverage ≥85% (`go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`)

## Out of Scope

- Integracion LLM/agent (Epic separado)
- v3 `.stem` (research separado — D3)
- Threshold configurability (hardcoded — D2)
- Marketplace distribution

## Features

| ID | Nombre | Descripcion |
|----|--------|-------------|
| F01 | [Body Content AST Infrastructure](F01-body-content-ast/) | goldmark integration y utilidades de extraccion de body |
| F02 | [Inference Category Expansion](F02-inference-detectors/) | Implementar categorias 5-13 como Go puro (engine portions) |
| F03 | [Analyze & Apply Commands](F03-analyze-apply-commands/) | Comandos CLI que orquestan inferencias y aplican resultados |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | Foundation: AST habilita body-aware categories |
| F02/S001 | — | Deterministic cats son independientes de AST |
| F02/S002 | F01 | Body-aware cats necesitan AST |
| F02/S003 | F01 | Semantic stubs usan body extraction |
| F03 | F02 | Analyze orquesta todas las categorias |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-03-03 | Solo engine, agent es Epic separado | computation-then-understanding: 2 capas, no mezclar |
| 2026-03-03 | goldmark como parser AST | Pure Go, 0 deps, habilita ~5% mejora en proporcion engine |
| 2026-03-03 | Cat 9/11 como stubs engine-only | 70-80% LLM — porciones Go extraen datos, agent decide |

## Gaps Activos

- S7: Report unidireccional (engine→agent) no probado — se validara cuando se implemente agent Epic
