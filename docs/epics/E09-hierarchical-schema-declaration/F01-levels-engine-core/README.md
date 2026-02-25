---
estado: Pending
tipo: feature
---
# F01: Levels Engine Core

**Epic**: [E09](../README.md)
**Satisface**: P1, P2, P3
**Objetivo**: El engine puede parsear `levels:` en `.stem` files, expandirlos a StemEntries virtuales, y detectar nesting violations
**Beneficio**: Habilita declarar schemas per-level en un solo `.stem` file, eliminando la proliferacion de child `.stem` repetitivos
**Milestone**: `go test ./internal/rules/ -run TestHierarchy` pasa con expansion correcta + nesting check

## Scope

**In**: HierarchyLevel struct, levels parsing, ExpandLevels(), ResolveForRecord(), CheckNesting(), levels map merge, unit tests
**Out**: Caller migration (F02), actual `.stem` file migration (F03), stemhealth checks, infer updates

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-level-parsing-and-expansion/) | Level Parsing and Expansion | El engine parsea `levels:` y genera StemEntries virtuales que producen effective schema correcto |
| [S002](S002-nesting-validation/) | Nesting Validation | `rootline validate` detecta records ubicados en niveles incorrectos de la jerarquia |

## Invariantes

- INV1 (heredado): Todos los tests existentes pasan sin modificacion
- INV2 (heredado): Coverage se mantiene >= 85%
- INV3: `.stem` files sin `levels:` producen el mismo resultado que antes (zero regression)

## Dependencias

- Ninguna — este Feature es foundation

## Fuente de verdad

- `internal/rules/rules.go` — StemFile struct
- `internal/rules/merge.go` — MergeStemFiles
- `internal/rules/validate.go` — Validate, conditionMatches
