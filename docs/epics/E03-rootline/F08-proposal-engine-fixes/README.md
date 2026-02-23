---
estado: Pending
tipo: feature
---
# F08: Proposal Engine Fixes

**Epic**: [E03](../README.md)
**Objetivo**: `rootline fix` genera proposals sin conflictos entre detectores
**Beneficio**: Elimina proposals duplicados/conflictivos que corrompen datos al aplicar `fix --all`
**Milestone**: `rootline fix --all --dry-run` desde repo root muestra todos los tipos de proposals sin duplicados ni conflictos entre detectores

## Scope

**In**: Corregir prioridad entre detectores en proposal engine, fix stem selection, agregar ScopeResolver a tree
**Out**: Nuevos tipos de proposals, refactors de API publica

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Fix Priority Conflicts](S001-fix-priority-conflicts/) | Proposals priorizados por tipo sin conflictos path/field |

## Fuente de verdad

- `internal/proposal/proposal.go` (Analyze function)
- `cmd/rootline/fix.go` (applyProposals, stem selection)
- `cmd/rootline/tree.go` (ScopeResolver)
