---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Add derive pipeline to fix --all

**Story**: [S001 Derive Pipeline in Fix](README.md)
**Contribuye a**: fix --all shows aggregate errors in output

## Preserva

- INV1: Existing fix proposals unchanged
  - Verificar: `go test ./internal/fix/ -race`

## Contexto

`runFixAll` in `cmd/rootline/fix.go` (L152-231) scans records and validates them but never runs the derive pipeline. This means `record.Derived` is always empty during fix, so aggregate computation never happens. The validation phase cannot detect aggregate mismatches because the computed values don't exist. Adding three derive calls after `index.Scan` and before the validation loop enables aggregate error detection.

## Alcance

**In**:
1. Add `"github.com/pablontiv/rootline/internal/derive"` to imports in `cmd/rootline/fix.go`
2. Add `derive.DeriveAllSimple(ctx, records, root)` after `index.Scan` (L171) and before validation loop (L180)
3. Add `derive.EnrichBuiltinsSimple(ctx, records, root)` after DeriveAllSimple
4. Add `derive.AggregateAllSimple(ctx, records, root)` after EnrichBuiltinsSimple

**Out**: Propagation logic, new proposal types, CLI flags

## Estado inicial esperado

- `cmd/rootline/fix.go` exists with `runFixAll` function
- `internal/derive` package has `DeriveAllSimple`, `EnrichBuiltinsSimple`, `AggregateAllSimple` functions
- `go build ./cmd/rootline/` compiles cleanly

## Criterios de Aceptacion

- `go build ./cmd/rootline/` compiles with derive import
- `go test ./... -race` passes (no regressions)
- `go vet ./cmd/rootline/` reports no issues

## Fuente de verdad

- `cmd/rootline/fix.go` — runFixAll function (L152-231)
- `internal/derive/aggregate.go` — AggregateAllSimple (L160-163)
- `internal/derive/record.go` — DeriveAllSimple, EnrichBuiltinsSimple
