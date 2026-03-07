---
estado: Completed
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Test that fix --all detects aggregate errors

**Story**: [S001 Derive Pipeline in Fix](README.md)
**Contribuye a**: fix --all shows aggregate errors in output

[[blocks:T001-add-derive-pipeline]]

## Preserva

- INV1: Existing fix proposals unchanged
  - Verificar: `go test ./internal/fix/ -race`

## Contexto

After T001 adds the derive pipeline to fix --all, we need to verify that aggregate errors are now visible in the fix output. The test creates a directory with a `.stem` defining an aggregate formula, a README with a stale `estado`, and child files with `estado: Completed`. The fix --all --dry-run should report the aggregate mismatch.

## Alcance

**In**:
1. Add test in `internal/e2e/fix_pipeline_test.go`
2. Test setup: temp dir with `.stem` (aggregate formula for estado), README.md with `estado: Pending`, child T001.md with `estado: Completed`
3. Run the fix pipeline (scan + derive + validate + propose)
4. Assert that validation errors include aggregate mismatch

**Out**: PropagateAggregate proposals (that's S002), E2E apply test (that's S003)

## Estado inicial esperado

- T001 completed: fix --all runs derive pipeline
- `internal/e2e/fix_pipeline_test.go` exists with existing test patterns

## Criterios de Aceptacion

- Test creates temp directory with stale aggregate parent
- After running derive + validate, aggregate error is detected
- `go test ./internal/e2e/ -run TestFixPipeline_DeriveInFix -race -v` passes

## Fuente de verdad

- `internal/e2e/fix_pipeline_test.go` — existing test patterns
- `cmd/rootline/fix.go` — runFixAll with derive pipeline
