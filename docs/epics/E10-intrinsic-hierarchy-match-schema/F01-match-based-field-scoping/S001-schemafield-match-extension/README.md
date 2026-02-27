# S001: SchemaField match extension

**Feature**: [F01 Match-based field scoping](../README.md)
**Capacidad**: SchemaField carries Match (glob pattern or list) and v2 .stem files parse correctly
**Cubre**: The data model and parsing side of the F01 milestone — fields can express their scope

## Antes / Despues

**Antes**: `SchemaField` has no `Match` field. Per-level schema is only available via `levels:` → `ExpandLevels`. There is no v2 stem format — all stems use the v1 format with `levels:` sections.

**Despues**: `SchemaField` carries a `Match` field supporting glob strings, lists of globs, or maps of pattern→config (for sequence id). `ParseStem` supports `version: 2` stems where fields use inline `match:` instead of `levels:`. v1 stems continue to parse unchanged.

## Criterios de Aceptacion (semanticos)

- [ ] A v2 `.stem` with `match:` on fields parses without error and produces correct SchemaField.Match values
- [ ] `required.match` scoping works — a field can be required only at certain levels
- [ ] v1 `.stem` files continue to parse identically (backward compat)

## Invariantes

- INV1: All existing workflows produce identical results for v1 stems
  - Verificar: `go test ./... -race` passes
- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-match-field-to-schemafield.md) | Add Match field to SchemaField struct with YAML unmarshaling |
| [T002](T002-implement-match-aware-field-resolution.md) | Implement match-aware field resolution function |
| [T003](T003-add-v2-stem-parsing.md) | Add v2 stem parsing with match syntax |

## Fuente de verdad

- `internal/rules/rules.go` — SchemaField struct definition
- `internal/rules/rules.go` — ParseStem function
- `docs/research/intrinsic-hierarchy-principle.md` — Part 3 (.stem v2 format example)
