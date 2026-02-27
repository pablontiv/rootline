---
estado: Completed
tipo: historia
---
# S003: Fix Proposal for Missing Aggregates

**Feature**: [F11 Aggregate Auto-Generation & Command Consolidation](../README.md)
**Capacidad**: `rootline fix` detecta stems jerárquicos sin aggregate para campos enum y propone agregarlos
**Cubre**: Detección y corrección automática de stems existentes sin aggregate

## Antes / Despues

**Antes**: `rootline fix` no detecta stems sin aggregate. Jerarquías con estado manual pasan sin warning. El usuario debe detectar y corregir manualmente.

**Despues**: `rootline fix --all --dry-run` reporta "would add aggregate for 'estado'" en stems que lo necesitan. `rootline fix --all` aplica la expresión generada via YAML AST sin corromper el .stem existente.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline fix --dry-run` detecta stem sin aggregate y propone AddAggregate
- [ ] `rootline fix` aplica AddAggregate al .stem via YAML AST

## Invariantes

- INV1: Tests existentes siguen pasando
  - Verificar: `go test ./cmd/rootline/ -run TestFix -v`
- INV2: fix no corrompe .stem existentes
  - Verificar: `rootline validate` después de fix

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-aggregate-proposal.md) | AddAggregate type en proposal engine + case en fix applier |

## Fuente de verdad

- `internal/proposal/proposal.go` — Analyze(), ProposalType enum
- `cmd/rootline/fix.go` — applyProposals, addEnumValueToNode pattern
- `internal/infer/hierarchy.go` — AnalyzeHierarchy
