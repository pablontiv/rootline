---
estado: Pending
tipo: feature
---
# F14: Sibling-Based Fix Inference

**Epic**: [E04](../README.md)
**Satisface**: P1 (fix propone valores correctos para campos enum faltantes), P2 (fix detecta outliers estadisticos)
**Objetivo**: `rootline fix --all` infiere valores correctos para campos enum faltantes o incorrectos usando la mayoria estadistica de siblings en el mismo directorio
**Beneficio**: Elimina la necesidad de corregir manualmente campos enum en batch — la herramienta propone el valor correcto basado en lo que ya existe en el directorio
**Milestone**: `rootline fix --all --dry-run` en un proyecto con campos faltantes/outliers produce propuestas `infer_from_siblings` y `correct_outlier` con valores correctos

## Scope

**In**: Nuevos proposal types (`infer_from_siblings`, `correct_outlier`), integracion en `Analyze()` pipeline, wiring en fix engine
**Out**: Inferencia basada en contenido/nombre de archivo, inferencia cross-directorio, cambios a .stem schema

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-sibling-inference-engine/) | Sibling Inference Engine | `rootline fix` usa mayoria estadistica de siblings para proponer valores de campos enum |

## Dependencias

- Ninguna — independiente del resto de E04

## Fuente de verdad

- `internal/proposal/proposal.go` — pipeline de analisis a extender
- `internal/fix/fix.go` — engine de aplicacion de propuestas
- `cmd/rootline/fix.go` — CLI fix command
