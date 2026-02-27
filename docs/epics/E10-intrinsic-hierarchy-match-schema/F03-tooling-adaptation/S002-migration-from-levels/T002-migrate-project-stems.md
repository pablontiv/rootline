---
estado: Specified
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T002: Migrate project .stem files to v2

**Story**: [S002 Migration from levels](README.md)
**Contribuye a**: All project .stem files use v2 format after migration

[[blocks:T001-implement-migrate-from-levels]]

## Preserva

- INV1: All existing workflows produce identical results
  - Verificar: `rootline validate docs/epics/` passes; `rootline validate --all docs/epics/` produces no new errors
- INV5: Migration preserves field semantics exactly
  - Verificar: Compare validate results before and after migration

## Contexto

The project has one `levels:`-based `.stem` file: `docs/epics/.stem` with 4 levels (epic, feature, story, task) containing per-level fields for `tipo`, `cliente`, `ejecutable_en`, `hold`, and sequence id. This task runs `rootline migrate --from-levels` on it and verifies the result.

## Especificacion Tecnica

1. Snapshot current state: `rootline validate --all docs/epics/ --output json > /tmp/before.json`
2. Run `rootline migrate --from-levels docs/epics/.stem`
3. Verify: `rootline validate --all docs/epics/ --output json > /tmp/after.json`
4. Compare before/after: same records, same errors, same field resolution
5. If any difference → investigate and fix migration logic or stem output

## Dependencias

- T001: migrate --from-levels command must work

## Alcance

**In**:
1. Run migration on `docs/epics/.stem`
2. Verify validation results unchanged
3. Verify describe output unchanged for sample records
4. Commit migrated .stem

**Out**: Migrating other projects' stems, removing v1 support

## Estado inicial esperado

- T001 completed: migrate --from-levels works
- `docs/epics/.stem` exists with v1 levels format

## Criterios de Aceptacion

- `docs/epics/.stem` is now v2 format (has `version: 2`, no `levels:`)
- `rootline validate --all docs/epics/` produces no new errors compared to before
- `rootline describe docs/epics/E10-intrinsic-hierarchy-match-schema/F01-match-based-field-scoping/` shows correct per-level fields
- `go test ./... -race` passes

## Fuente de verdad

- `docs/epics/.stem` — Target stem file
- `docs/epics/` — All records under the stem
