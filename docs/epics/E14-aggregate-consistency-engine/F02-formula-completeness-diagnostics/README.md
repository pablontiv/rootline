---
estado: Pending
tipo: feature
---
# F02: Formula Completeness Diagnostics

**Epic**: [E14 Aggregate Consistency Engine](../README.md)
**Satisface**: P2
**Objetivo**: Detect incomplete aggregate formulas at schema validation time
**Beneficio**: Prevents silent incorrect computation when new enum values appear
**Milestone**: `rootline validate --all --stem-health` warns about formulas that don't cover all enum values

## Scope

**In**: Stem-health diagnostic that compares aggregate expression string literals against enum values
**Out**: Auto-fixing formulas, formula migration tooling

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Stem-Health Formula Check](S001-stem-health-formula-check/) | Stem-health check warns about incomplete aggregate formulas |

## Invariantes

- INV1 (heredado): `go test ./... -race` pasa verde
- INV2 (heredado): Existing stem-health checks unchanged

## Dependencias

- F01 must be complete first (formula check supplements propagation pre-check)

## Fuente de verdad

- `internal/rules/stem_health.go` — existing 7 stem-health checks
- `docs/epics/.stem` — aggregate expressions
