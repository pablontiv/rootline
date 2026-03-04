---
estado: Specified
tipo: historia
---
# S002: Incremental Analysis Mode

**Feature**: [F03 Analyze & Apply Commands](../README.md)
**Capacidad**: `rootline analyze --incremental` detecta delta entre .stem actual y datos
**Cubre**: Postcondicion P3 del Epic

## Antes / Despues

**Antes**: `rootline analyze` (de S001) ejecuta analisis completo cada vez. No hay forma de detectar solo lo que cambio desde el ultimo .stem update.

**Despues**: `rootline analyze --incremental` compara inferencias contra .stem actual y reporta solo deltas: campos inferidos que no estan en .stem, reglas inferidas que difieren de las existentes.

## Criterios de Aceptacion (semanticos)

- [ ] `--incremental` flag existe en analyze command
- [ ] Con .stem que ya tiene `required: [estado]` y datos donde estado es 100% presente → no produce inferencia (ya cubierto)
- [ ] Con .stem sin `required:` y datos donde estado es 100% → produce inferencia delta
- [ ] Report incremental solo incluye inferencias no cubiertas por .stem
- [ ] `go test ./... -race` pasa verde

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-delta-detection.md) | Implementar delta detection entre .stem e inferencias |
| [T002](T002-incremental-flag.md) | Añadir --incremental flag a analyze |
| [T003](T003-incremental-tests.md) | Tests de analisis incremental |

## Fuente de verdad

- `internal/rules/drift.go` — DetectDrift (referencia: drift detection existente)
- `internal/infer/infer.go` — Analyze()
- `internal/rules/rules.go` — StemSchema para comparacion
