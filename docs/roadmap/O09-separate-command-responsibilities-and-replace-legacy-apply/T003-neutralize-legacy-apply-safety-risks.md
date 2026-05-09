---
estado: Completed
tipo: task
---
# T003: Neutralize legacy apply safety risks

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T002-add-apply-safety-characterization-tests.md]]

## Preserva

- INV1: `.stem` mutation is explicit schema evolution, never a hidden side effect of data repair.
  - Verificar: `apply --dry-run` never writes `.stem` while legacy compatibility exists.
- INV2: Every machine-readable command used by agents emits parseable, versioned JSON.
  - Verificar: JSON-mode apply tests parse stdout.

## Contexto

This is a safety patch for the legacy command, not the final architecture. The command currently scaffolds `.stem` files before respecting dry-run and sends schema inferences to an applier that always writes.

## Alcance

**In**:
1. Make legacy `apply --dry-run` a true no-write operation for schema updates and `missing_schema` scaffold.
2. Move scaffold status from direct stdout prints into structured result data so JSON stays parseable.
3. Preserve non-dry-run legacy behavior unless explicitly deprecated by T011.
4. Keep the fix surgical; do not redesign target routing or proposal taxonomy in this task.

**Out**:
- Full transactionality.
- New `schema apply` or `repair apply` commands.
- Monotonic `.stem` semantics.

## Estado inicial esperado

- T002 characterization tests exist and show the current failure modes.

## Criterios de Aceptación

- `apply --dry-run` does not create or modify `.stem` files.
- JSON output from `apply` is parseable in scaffold scenarios.
- Existing non-dry-run apply tests still pass unless intentionally marked deprecated.
- Focused tests and `go test ./cmd/rootline ./internal/infer ./internal/e2e -run Apply` pass.

## Fuente de verdad

- `cmd/rootline/apply.go`
- `internal/infer/apply.go`
- `internal/infer/scaffold.go`
- `internal/infer/apply_test.go`
- `internal/e2e/apply_test.go`
