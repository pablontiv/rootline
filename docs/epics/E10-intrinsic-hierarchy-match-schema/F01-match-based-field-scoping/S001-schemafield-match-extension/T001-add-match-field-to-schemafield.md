---
estado: Specified
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T001: Add Match field to SchemaField struct

**Story**: [S001 SchemaField match extension](README.md)
**Contribuye a**: v2 `.stem` with `match:` on fields parses without error and produces correct SchemaField.Match values

## Preserva

- INV1: All existing workflows produce identical results for v1 stems
  - Verificar: `go test ./... -race` passes
- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

`SchemaField` in `internal/rules/rules.go` currently has no way to express which directory patterns a field applies to. The v2 stem format needs fields to carry `match:` metadata. The `Match` field must support three forms from the research document: a single glob string (`match: "T*"`), a list of globs (`match: ["F*", "T*"]`), or a map of pattern→config (`match: {"E*": {prefix: E, digits: 2}}`). The map form is needed for sequence `id` fields where prefix/digits vary by level.

## Especificacion Tecnica

Add to `SchemaField` struct in `internal/rules/rules.go`:
- `Match` field with a custom type (e.g., `FieldMatch`) that implements `yaml.Unmarshaler`
- `FieldMatch` should parse: string → single pattern, []string → multiple patterns, map[string]interface{} → pattern-to-config map
- Add `RequiredMatch` field (or sub-struct) for `required.match` scoping — a field can be required only when the record matches certain patterns
- Ensure JSON serialization works for `--output json` commands

## Dependencias

- None — this is the first task in the chain

## Alcance

**In**:
1. Add `FieldMatch` type with `UnmarshalYAML` supporting string, list, and map forms
2. Add `Match` field to `SchemaField`
3. Add `RequiredMatch` field or extend `Required` to support match-scoped requirement
4. Add unit tests for YAML unmarshaling of all three forms
5. Ensure existing `SchemaField` tests still pass

**Out**: Resolution logic (T002), v2 parsing (T003), anything outside `rules.go`

## Estado inicial esperado

- `internal/rules/rules.go` exists with current `SchemaField` struct
- No `Match` field exists on `SchemaField`

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestFieldMatch` passes — tests unmarshaling of string, list, and map forms
- `go test ./internal/rules/ -run TestParseStem` still passes — v1 stems unchanged
- `go vet ./...` and `golangci-lint run ./...` pass
- `SchemaField` with `Match: nil` behaves identically to current (no filtering)

## Fuente de verdad

- `internal/rules/rules.go` — SchemaField struct (lines 24-35)
- `docs/research/intrinsic-hierarchy-principle.md` — Part 3, v2 format example (lines 228-251)
