---
estado: Completed
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T003: Add v2 stem parsing with match syntax

**Story**: [S001 SchemaField match extension](README.md)
**Contribuye a**: v2 `.stem` with `match:` on fields parses without error

[[blocks:T001-add-match-field-to-schemafield]]

## Preserva

- INV1: All existing workflows produce identical results for v1 stems
  - Verificar: `go test ./... -race` passes

## Contexto

`ParseStem` in `internal/rules/rules.go` currently parses all stems as v1 (the `Version` field exists but isn't used for branching). When `version: 2` is present, parsing should expect fields with inline `match:` and reject `levels:` sections. When `version: 1` (or unset), parsing behaves as today. This task adds the version-aware branching and v2-specific validation.

## Especificacion Tecnica

Modify `ParseStem` in `internal/rules/rules.go`:
- After unmarshaling, check `stem.Version`
- If version == 2: verify `stem.Levels` is nil/empty (reject v2 + levels combo), apply source tagging and severity defaulting to fields including their Match metadata
- If version == 1 or 0 (unset): current behavior unchanged
- Add validation: v2 stems must not have `levels:` section
- Ensure `SchemaField.Match` is populated correctly from YAML for v2 stems

## Dependencias

- T001: SchemaField.Match must exist for YAML unmarshaling to populate it

## Alcance

**In**:
1. Add version-aware branching in `ParseStem`
2. Reject `levels:` in v2 stems with clear error message
3. Ensure Match fields are correctly populated from v2 YAML
4. Add tests for v2 stem parsing
5. Verify v1 parsing unchanged

**Out**: Resolution logic (T002), ResolveForRecord changes (S002)

## Estado inicial esperado

- T001 completed: SchemaField has Match field
- `ParseStem` exists with v1 parsing logic

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestParseStemV2` passes — v2 stem with match fields parses correctly
- v2 stem with `levels:` section returns a parse error
- v1 stems parse identically to before (backward compat)
- `go vet ./...` passes

## Fuente de verdad

- `internal/rules/rules.go` — ParseStem function (lines 157-182)
- `docs/research/intrinsic-hierarchy-principle.md` — Part 3, v2 format example
