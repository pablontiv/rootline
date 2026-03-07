---
estado: Pending
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: E2E round-trip test for aggregate propagation

**Story**: [S003 Fix Pipeline Integration](README.md)
**Contribuye a**: Deep hierarchy propagates bottom-up correctly; validate shows 0 errors after fix

[[blocks:T001-wire-propagate-to-fix]]

## Preserva

- INV1: Existing proposals unchanged
  - Verificar: `go test ./internal/fix/ -race`

## Contexto

This is the end-to-end validation that the entire propagation pipeline works. Two test scenarios: (1) single level — one README with stale estado, children all Completed, fix corrects it. (2) Three levels (Epic->Feature->Story->Task) — all tasks Completed, fix propagates bottom-up through all intermediate READMEs.

## Alcance

**In**:
1. Add tests in `internal/e2e/fix_pipeline_test.go`
2. Test 1 (single level): temp dir with `.stem` (aggregate), README.md `estado: Pending`, T001.md `estado: Completed`. Run full fix pipeline. Re-read README -> assert `estado: Completed`. Re-validate -> 0 aggregate errors.
3. Test 2 (deep hierarchy): Epic/Feature/Story/Task structure. All tasks Completed. Run fix. Verify all READMEs at Story, Feature, Epic levels are updated to Completed.

**Out**: Unit tests (S002), CLI flag tests

## Estado inicial esperado

- T001 (wire) completed: fix --all applies PropagateAggregate proposals
- `internal/e2e/fix_pipeline_test.go` has existing round-trip test patterns

## Criterios de Aceptacion

- Single-level test: stale README corrected after fix, re-validate clean
- Deep hierarchy test: all 3 intermediate READMEs updated bottom-up
- `go test ./internal/e2e/ -run TestFixPipeline_Propagate -race -v` passes

## Fuente de verdad

- `internal/e2e/fix_pipeline_test.go` — existing test patterns
- `cmd/rootline/fix.go` — runFixAll pipeline
