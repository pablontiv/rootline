---
estado: Completado
tipo: historia
cliente: Platform Owner
---
# S002: Bulk Operations

**Feature**: [F01 Migration Tooling](../README.md)
**Capacidad**: rootline migrate --rename actualiza un campo en todos los documentos afectados y mantiene un log de migraciones

## Antes / Despues

**Antes**: Renombrar un campo (ej: "titulo" → "title") requiere editar N archivos manualmente o un script ad-hoc. El .stem tambien necesita actualizarse. No hay registro de que cambio.

**Despues**: `rootline migrate --rename titulo=title docs/epics/` actualiza frontmatter en todos los archivos que tienen el campo, actualiza el .stem, y registra la operacion en `.rootline-migrations`.

## Criterios de Aceptacion (semanticos)

- [ ] rootline migrate --rename actualiza campo en frontmatter de todos los archivos
- [ ] .stem schema se actualiza con el nuevo nombre de campo
- [ ] Migration log registra cada operacion
- [ ] --dry-run muestra que se haria sin modificar

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-migrate-rename.md) | Implementar rootline migrate --rename |
| [T002](T002-migration-log.md) | Implementar migration log |

## Fuente de verdad

- `internal/proposal/proposal.go` (rewriteFrontmatter pattern)
- `cmd/rootline/fix.go` (addEnumValueToNode — AST modification pattern)
