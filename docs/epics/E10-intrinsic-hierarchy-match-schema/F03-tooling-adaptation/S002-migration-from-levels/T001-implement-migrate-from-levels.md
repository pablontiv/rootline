---
estado: Completed
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T001: Implement migrate --from-levels command

**Story**: [S002 Migration from levels](README.md)
**Contribuye a**: `rootline migrate --from-levels` converts v1 stems to semantically equivalent v2

## Preserva

- INV5: Migration preserves field semantics exactly
  - Verificar: Before/after validation produces identical results for all records under the migrated .stem

## Contexto

Users with existing v1 `.stem` files containing `levels:` need an automated migration path to v2. The `internal/migrate/` package already has migration infrastructure (diff detection, bulk rename, split). This task adds a `--from-levels` flag that reads a v1 `.stem`, extracts per-level field definitions from the `levels:` map, converts them to v2 `match:`-based schema fields, and writes the result.

## Especificacion Tecnica

New function in `internal/migrate/` (e.g., `levels_to_match.go`):
- `ConvertLevelsToMatch(stem *StemFile) (*StemFile, error)`
- For each level in `stem.Levels`: extract `Match` pattern and level-specific `Schema` fields
- For each level-specific field: set `field.Match` to the level's pattern
- Merge into root schema (handling conflicts: if same field in multiple levels, combine match patterns)
- Handle sequence id: if levels have different prefix/digits, use map-form match
- Set `version: 2`, clear `Levels`

New CLI flag in `cmd/rootline/migrate.go`:
- `--from-levels <path>` flag
- Read .stem file, call ConvertLevelsToMatch, write result

## Dependencias

- F01 completed: SchemaField.Match and v2 parsing must work

## Alcance

**In**:
1. Implement ConvertLevelsToMatch function
2. Add --from-levels flag to migrate command
3. Handle all levels field types (schema, sequence, validate rules)
4. Write result as v2 .stem YAML
5. Unit tests for conversion

**Out**: Actually migrating project stems (T002), removing v1 support (F04)

## Estado inicial esperado

- F01 completed: v2 stem format works
- `internal/migrate/` exists with migration infrastructure
- `cmd/rootline/migrate.go` exists

## Criterios de Aceptacion

- `go test ./internal/migrate/ -run TestConvertLevelsToMatch` passes
- Conversion of the sample v1 stem from the research doc produces correct v2 output
- `rootline migrate --from-levels <test-stem>` writes a valid v2 .stem file
- Migrated stem resolves identically to original for test records
- `go test ./... -race` passes

## Fuente de verdad

- `internal/migrate/` — Migration engine
- `internal/rules/rules.go` — StemFile, HierarchyLevel, SchemaField
- `cmd/rootline/migrate.go` — Migrate command
- `docs/research/intrinsic-hierarchy-principle.md` — Part 3 (before/after .stem example)
