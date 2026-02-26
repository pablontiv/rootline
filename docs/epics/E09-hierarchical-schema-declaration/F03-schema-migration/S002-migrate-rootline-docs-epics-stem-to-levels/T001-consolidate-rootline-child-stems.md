---
estado: Completed
tipo: documentation
ejecutable_en: 1 sesion
---
# T001: Consolidate rootline child .stem files into root levels and validate

**Story**: [S002 Migrate rootline docs/epics .stem to levels](README.md)
**Contribuye a**: rootline/docs/epics consolida ~70 child `.stem` en root `.stem` con levels

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV5: Todos los documentos existentes siguen validando correctamente
  - Verificar: `rootline validate --all docs/epics/`

## Contexto

Rootline's own `docs/epics/` has a hierarchical `.stem` structure with ~70 child `.stem` files:

- `docs/epics/.stem` (root): `id: {sequence, E, 2}` + derive/aggregate/links/structural
- `docs/epics/E03-rootline/.stem` (feature level): `id: {sequence, F, 2}`, `tipo: {enum, warn}`
- `docs/epics/E03-rootline/F05-*/.stem` (story level): `id: {sequence, S, 3}`, `cliente: {string}`
- `docs/epics/E03-rootline/F05-*/S001-*/.stem` (task level): `id: {sequence, T, 3}`, `ejecutable_en`, `hold`

Most child `.stem` files are nearly identical — they repeat the same pattern of `id: {sequence}` plus optional fields at each level. These can be consolidated into a single root `.stem` with `levels:`.

**Important**: Some child `.stem` files have genuine overrides (e.g., `cliente` only at story level in F05). These should either be absorbed into the levels definition or kept as real child `.stem` overrides.

## Dependencias

- F01 + F02: Levels engine and caller migration complete
- F03/S001/T001: Homeserver migration done first (establishes the pattern)

## Alcance

**In**:
1. Audit all child `.stem` files under `docs/epics/` to identify common vs unique fields
2. Rewrite `docs/epics/.stem` with `levels:` section covering 4 levels
3. Keep derive, aggregate, links, structural in the base (not per-level)
4. Delete child `.stem` files that are fully covered by levels
5. Keep child `.stem` files with genuine overrides not representable in levels
6. Run `rootline validate --all docs/epics/` to verify no regressions
7. Run `go test ./... -race` to verify tests pass

**Out**: Engine changes, homeserver migration (separate task)

## Estado inicial esperado

- F01, F02, F03/S001 complete
- `docs/epics/.stem` exists with derive/aggregate/links/structural
- ~70 child `.stem` files exist in subdirectories

## Criterios de Aceptacion

- `docs/epics/.stem` has `levels:` section with 4 levels (epic, feature, story, task)
- Most child `.stem` files deleted (only genuine overrides remain)
- `rootline validate --all docs/epics/` passes with 0 errors
- `rootline describe docs/epics/E09-*/F01-*/S001-*/ --field schema` shows correct per-level fields
- `go test ./... -race` passes
- Effective schema per-level is identical to the previous child `.stem` chain

## Fuente de verdad

- `docs/epics/.stem` — root stem to rewrite
- `docs/epics/E03-rootline/.stem` — feature level example
- `docs/epics/E03-rootline/F05-mcp-distribution/.stem` — story level with `cliente` override
- `docs/epics/E03-rootline/F05-mcp-distribution/S001-mcp-server/.stem` — task level example
