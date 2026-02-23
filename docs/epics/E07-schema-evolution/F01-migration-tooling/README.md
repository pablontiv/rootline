---
estado: Pending
tipo: feature
---
# F01: Migration Tooling

**Epic**: [E07](../README.md)
**Objetivo**: rootline detecta breaking changes en .stem y migra documentos automaticamente
**Beneficio**: Schema evolution es segura y auditada. Renombrar un campo no requiere editar N archivos manualmente.
**Milestone**: `rootline migrate --dry-run` lista breaking changes; `rootline migrate --rename old=new` actualiza todos los archivos afectados

## Scope

**In**: Schema diff, breaking change detection, field rename, migration log
**Out**: Interactive migration UI, rollback, multi-repo federation, .stem versioning

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Change Detection](S001-change-detection/) | rootline migrate --dry-run muestra que cambio y que se rompe |
| S002 | [Bulk Operations](S002-bulk-operations/) | rootline migrate --rename actualiza campo en todos los archivos |

## Dependencias

- internal/rules/ (StemFile parsing — existente)
- internal/proposal/ (fix patterns — reusable para rewrite)

## Fuente de verdad

- `internal/rules/rules.go` (StemFile struct)
- `internal/proposal/proposal.go` (rewriteFrontmatter pattern)
- `cmd/rootline/fix.go` (CLI pattern para dry-run + apply)
