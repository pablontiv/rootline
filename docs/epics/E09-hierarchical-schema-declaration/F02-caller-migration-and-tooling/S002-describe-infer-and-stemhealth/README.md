---
estado: Specified
---
# S002: Describe, Infer and Stemhealth

**Feature**: [F02 Caller Migration and Tooling](../README.md)
**Capacidad**: stemhealth detecta levels malformados e infer genera formato levels
**Cubre**: Completar tooling para que levels sea first-class

## Antes / Despues

**Antes**: `stemhealth` no sabe de `levels:` — no detecta children references invalidos ni levels circulares. `infer --hierarchy` genera child `.stem` files separados.

**Despues**: `stemhealth` incluye checks para levels (valid children refs, no cycles). `infer --hierarchy` genera un solo `.stem` con `levels:` en vez de multiples child files.

## Criterios de Aceptacion (semanticos)

- [ ] stemhealth detecta children que referencian levels inexistentes
- [ ] stemhealth detecta cycles en children graph
- [ ] `infer --hierarchy` genera `.stem` con `levels:` section
- [ ] Ambos pasan sus test suites

## Invariantes

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-stemhealth-checks-for-levels.md) | Add stemhealth checks for levels validity |
| [T002](T002-update-infer-hierarchy-to-levels.md) | Update infer --hierarchy to generate levels format |
| [T003](T003-fix-describe-levels-expansion.md) | Fix describe to expand levels for file targets |

## Fuente de verdad

- `internal/rules/stemhealth.go` — existing 7 health checks
- `internal/infer/hierarchy.go` — hierarchy inference
