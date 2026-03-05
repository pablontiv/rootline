---
estado: Completed
tipo: feature
---
# F03: Analyze & Apply Commands

**Epic**: [E13 Inference Engine](../README.md)
**Satisface**: P1, P2, P3
**Objetivo**: Comandos CLI que orquestan inferencias y aplican resultados
**Beneficio**: Interfaz unificada para ejecutar y aplicar todos los detectores de inferencia
**Milestone**: `rootline analyze` genera report JSON; `rootline apply` ejecuta cambios; `--incremental` detecta delta

## Scope

**In**: Comando analyze, report JSON schema, comando apply, modo incremental
**Out**: Análisis semántico (agent consume el report, Epic separado)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Analyze Command & Report Format](S001-analyze-command/) | `rootline analyze` orquesta detectores y genera report |
| S002 | [Incremental Analysis Mode](S002-incremental-analysis/) | `--incremental` detecta delta entre .stem y datos |
| S003 | [Apply Command](S003-apply-command/) | `rootline apply` ejecuta inferencias aprobadas del report |

## Invariantes

- INV1 (heredado): `go test ./... -race` pasa verde en cada commit
- INV2 (heredado): Contratos JSON mantienen `"version": 1`
- INV3 (heredado): Coverage ≥85%

## Dependencias

- Depende de F02 (detectores deben existir para orquestarlos)

## Fuente de verdad

- `cmd/rootline/` — subcomandos CLI (analyze.go, apply.go se crean aqui)
- `internal/infer/` — detectores de inferencia
- `internal/proposal/` — interfaz engine→agent (proposal struct)
