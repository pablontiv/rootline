---
estado: Completed
---
# S001: Sibling Inference Engine

**Feature**: [F14 Sibling-Based Fix Inference](../README.md)
**Capacidad**: `rootline fix` propone valores de campos enum basados en la mayoria de siblings en el mismo directorio
**Cubre**: Milestone de F14 — fix produce propuestas infer_from_siblings y correct_outlier

## Antes / Despues

**Antes**: `rootline fix` agrega campos enum faltantes con el primer valor del enum (ej: `servicio-docker`) — incorrecto en la mayoria de casos. Valores validos pero semanticamente incorrectos (ej: `documentation` en deploy tasks) no se detectan.

**Despues**: `rootline fix` agrupa records por directorio, calcula la mayoria estadistica, y propone el valor correcto. Outliers (valores validos pero diferentes de la mayoria) se detectan y reportan.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline fix --all --dry-run` propone `infer_from_siblings` cuando mayoria de siblings tiene el mismo valor
- [ ] `rootline fix --all --dry-run` propone `correct_outlier` cuando un record difiere de fuerte consenso
- [ ] Ninguna propuesta generada para campos no-enum
- [ ] README files excluidos del grouping de siblings

## Invariantes

- INV1: Tests existentes pasan sin cambios
  - Verificar: `go test ./... -race`
- INV2: Coverage >= 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
- INV3: `infer_from_siblings` tiene prioridad sobre `add_field` pero no sobre `extract_body` ni `infer_from_children`
  - Verificar: test de integracion en Analyze pipeline

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-implement-sibling-detectors.md) | Implementar detectInferFromSiblings, detectOutlierValue, y majorityValue con tests unitarios |
| [T002](T002-integrate-pipeline-and-fix.md) | Integrar en Analyze pipeline, fix engine, y CLI output |

## Fuente de verdad

- `internal/proposal/proposal.go` — Analyze pipeline
- `internal/fix/fix.go` — ApplyProposals switch
- `cmd/rootline/fix.go` — proposalsToFixResults
