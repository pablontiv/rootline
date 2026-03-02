---
estado: Specified
tipo: feature
---
# F01: Hard-reject V1 Stems

**Epic**: [E12 V1 Stem Removal](../README.md)
**Satisface**: P1, P3
**Objetivo**: El engine rechaza stems v1 en tiempo de carga con error explicativo
**Beneficio**: Elimina branches condicionales v1/v2 en rules.go, hierarchy.go, stemhealth.go
**Milestone**: `rootline validate` con stem version 0 o 1 produce error; todos los tests usan v2

## Scope

**In**: Rechazo en ParseStem, migracion de test stems, limpieza de health check v1
**Out**: Eliminar codigo de migracion (eso es F02)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Engine rejects v1](S001-engine-rejects-v1/) | Stems v1 producen error en parse time |

## Invariantes

- INV1 (heredado): `go test ./... -race` pasa verde en cada commit
- INV2 (heredado): Coverage ≥85%

## Dependencias

- Ninguna (este Feature es foundation)

## Fuente de verdad

- `internal/rules/rules.go` — ParseStem, rejectLevelsInV2
- `internal/rules/hierarchy.go` — branch v1/v2
- `internal/rules/stemhealth.go` — check version-deprecated
- `internal/rules/testdata/*.stem` — 5 test stem files
- `cmd/rootline/*_test.go`, `internal/rules/*_test.go` — inline test stems
