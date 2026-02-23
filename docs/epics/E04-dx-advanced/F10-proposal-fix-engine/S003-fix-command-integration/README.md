---
tipo: historia
cliente: Platform Owner
---
# S003: Fix Command Integration

**Feature**: [F10 Proposal-Based Fix Engine](../README.md)
**Capacidad**: `rootline fix --dry-run` muestra propuestas categorizadas con explicaciones, y `rootline fix` las aplica incluyendo modificaciones al .stem

## Antes / Despues

**Antes**: `rootline fix --dry-run` muestra cambios planos (`add estado="Pending"`, `correct estado: "Obsoleto" -> "Completado"`) sin contexto ni categorizacion. `rootline fix` solo modifica archivos de datos, nunca el .stem. Las correcciones son lossy.

**Despues**: `rootline fix --dry-run` muestra un Report con propuestas agrupadas por tipo (extend_enum, migrate_value, extract_body, infer_from_children), cada una con descripcion, razon, y archivos afectados. `rootline fix` aplica las propuestas en orden de prioridad, incluyendo modificaciones al .stem (extend_enum agrega valores al enum).

## Criterios de Aceptacion (semanticos)

- [ ] `rootline fix --all --dry-run` produce JSON con `kind: "rootline/fix-proposals"`
- [ ] `rootline fix --all --dry-run -o table` muestra tabla legible con Type, Description, Files, Resolves
- [ ] `rootline fix --all` seguido de `rootline validate --all` produce 0 errores
- [ ] `go build ./cmd/rootline/` compila sin errores

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-wire-proposals-into-fix-dry-run.md) | Integrar proposal.Analyze() en fix, cambiar output de dry-run |
| [T002](T002-apply-proposals-with-stem-rewrite.md) | Aplicar propuestas incluyendo rewrite de .stem |

## Fuente de verdad

- `cmd/rootline/fix.go` — archivo a modificar
- `internal/proposal/` — package de S002
