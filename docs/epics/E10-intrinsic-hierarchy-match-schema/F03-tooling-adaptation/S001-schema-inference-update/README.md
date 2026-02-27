# S001: Schema inference update

**Feature**: [F03 Tooling adaptation](../README.md)
**Capacidad**: Infer and init produce v2 match-based schemas instead of levels
**Cubre**: The schema generation side of the F03 milestone

## Antes / Despues

**Antes**: `infer.ToLevelsMap()` generates a `levels:` map structure. `cmd/rootline/init.go` calls `generateHierarchicalRootYAML` which embeds `levels:` in the output `.stem` file. All generated schemas use the v1 format.

**Despues**: Infer produces v2 format with per-field `match:` entries. Root-level fields (present at all levels) have no `match`, per-level fields get `match: ["E*"]` etc. `rootline init` generates v2 stems with `version: 2`.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline init` on a hierarchical directory produces a v2 `.stem` with `match:` fields
- [ ] Inferred schemas preserve field distribution logic (root vs per-level fields)
- [ ] Generated v2 stems pass `rootline validate`

## Invariantes

- INV2: Code coverage stays above 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
- INV5: Migration preserves field semantics exactly
  - Verificar: Generated v2 stem resolves identically to manually equivalent v1 stem

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-replace-tolevelsmap.md) | Replace ToLevelsMap with match-based output |
| [T002](T002-update-init-command.md) | Update init command to generate v2 stems |

## Fuente de verdad

- `internal/infer/hierarchy.go` — ToLevelsMap, distributeFields, HierarchyResult
- `cmd/rootline/init.go` — buildHierarchicalStems, generateHierarchicalRootYAML
- `cmd/rootline/init_test.go` — Init command tests
