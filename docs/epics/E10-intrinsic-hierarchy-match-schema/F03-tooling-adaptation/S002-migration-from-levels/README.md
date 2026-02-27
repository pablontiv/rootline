# S002: Migration from levels

**Feature**: [F03 Tooling adaptation](../README.md)
**Capacidad**: Automated v1 (levels) to v2 (match) .stem conversion
**Cubre**: The migration tooling side of the F03 milestone

## Antes / Despues

**Antes**: No automated conversion from v1 `levels:` stems to v2 match-based format. Users with existing `levels:` stems must manually rewrite them.

**Despues**: `rootline migrate --from-levels <path>` reads a v1 `.stem` with `levels:`, extracts per-level fields and patterns, converts to v2 with inline `match:` on fields, and writes the result. The project's own `docs/epics/.stem` is migrated to v2.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline migrate --from-levels` converts v1 stems to semantically equivalent v2
- [ ] All project .stem files use v2 format after migration
- [ ] `rootline validate docs/epics/` passes after migration

## Invariantes

- INV5: Migration preserves field semantics exactly
  - Verificar: Validate all docs/epics/ records before and after, compare results

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-implement-migrate-from-levels.md) | Implement migrate --from-levels command |
| [T002](T002-migrate-project-stems.md) | Migrate project .stem files to v2 |

## Fuente de verdad

- `internal/migrate/` — Migration engine, split.go (reference for hierarchical approach)
- `cmd/rootline/migrate.go` — Migrate command
- `docs/epics/.stem` — Target for migration
