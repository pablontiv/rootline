---
estado: Specified
tipo: task
---
# T002: Add apply safety characterization tests

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T001-codify-command-responsibility-contracts.md]]

## Preserva

- INV2: Every machine-readable command used by agents emits parseable, versioned JSON.
  - Verificar: tests parse JSON stdout for `apply` scenarios.

## Contexto

Current coverage calls `apply --dry-run` but does not assert that files remain unchanged. The confirmed bugs include schema writes during dry-run, `missing_schema` scaffold creation during dry-run, human `scaffolded ...` stdout contaminating JSON, root-most `.stem` target selection, and non-transactional partial writes.

## Alcance

**In**:
1. Add CLI tests for `apply --dry-run` with schema inferences proving `.stem` bytes are unchanged.
2. Add CLI tests for `apply --dry-run` with `missing_schema` proving no `.stem` is created.
3. Add CLI tests proving default/JSON stdout is parseable when `missing_schema` is present.
4. Add nested `.stem` fixture proving the current target-selection behavior is characterized.
5. Add partial-write characterization for scaffold/schema/data sequence, even if it is marked as known failing until the fix task.

**Out**:
- Implementing fixes.
- Designing the replacement commands.

## Estado inicial esperado

- `cmd/rootline/coverage_test.go` has weak apply coverage.
- `internal/infer/apply_test.go` has dry-run coverage for data corrections only.

## Criterios de Aceptación

- Tests fail against the current unsafe behavior before fixes are applied.
- Tests identify file paths and bytes that must remain unchanged for dry-run.
- Tests are focused enough to guide T003 without requiring the full command redesign.
- `go test ./cmd/rootline ./internal/infer -run 'Apply|Scaffold'` exercises the new cases.

## Fuente de verdad

- `cmd/rootline/apply.go`
- `cmd/rootline/coverage_test.go`
- `internal/infer/apply.go`
- `internal/infer/scaffold.go`
- `internal/infer/apply_test.go`
- `internal/e2e/apply_test.go`
