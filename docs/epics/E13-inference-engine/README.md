---
estado: Completed
tipo: feature
---
# E13: Inference Engine

**Metrica de exito**: `rootline analyze <path>` produce JSON report cubriendo todos los detectores de inferencia; `rootline apply` ejecuta inferencias aprobadas
**Timeline**: 2026-Q1

## Intencion

Rootline tiene un schema inference engine parcial (`internal/infer/`) que detecta tipos, enums, sequences, y aggregates. Este Epic extiende el engine con 9 detectores nuevos (link-type validation, body section analysis, back-reference consistency, constant field detection, formal dependency extraction, cross-reference validation, traceability link extraction, invariant extraction, sub-schema detection), añade goldmark como parser AST para body content, y expone los resultados via los comandos `rootline analyze` y `rootline apply`.

La arquitectura sigue el modelo de 2 capas: engine (Go) + agent (LLM). Este Epic cubre solo el engine — los detectores Go. Los detectores que requieren análisis semántico (formal dependency, traceability) producen datos parciales que un futuro Epic de integración agent completará.

## Postcondiciones

- P1: `rootline analyze <path>` produce JSON report con `version: 1` cubriendo todos los detectores
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
| F02 | [Inference Detectors](F02-inference-detectors/) | Implementar 9 detectores nuevos de inferencia en Go |
| F03 | [Analyze & Apply Commands](F03-analyze-apply-commands/) | Comandos CLI que orquestan inferencias y aplican resultados |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | Foundation: AST habilita body-aware categories |
| F02/S001 | — | Structural detectors son independientes de AST |
| F02/S002 | F01 | Body-aware detectors necesitan AST |
| F02/S003 | F01 | Semantic extraction usa body extraction |
| F03 | F02 | Analyze orquesta todos los detectores |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-03-03 | Solo engine, agent es Epic separado | 2 capas: engine detecta, agent razona — no mezclar |
| 2026-03-03 | goldmark como parser AST | Pure Go, 0 deps, habilita ~5% mejora en proporcion engine |
| 2026-03-03 | Dependency/traceability como stubs engine-only | Detectores extraen datos formales, agent hace disambiguation |

## Gaps Activos

- S7: Report unidireccional (engine→agent) no probado — se validara cuando se implemente agent Epic
