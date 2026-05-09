---
estado: Completed
tipo: task
---
# T004: Upgrade stem health monotonic diagnostics

**Outcome**: [O10 Move .stem to monotonic hierarchical constraints](README.md)
**Contribuye a**: CE3 y CE4 del Outcome.

[[blocked_by:./T002-implement-layered-stem-resolver.md]]

## Preserva

- INV1: Moving down the directory tree must never silently reduce parent guarantees.
  - Verificar: stem-health fails loosening/widening/removal under monotonic mode.

## Contexto

Current stem-health has `field-override` warnings and type-consistency failures, but it does not distinguish valid narrowing from destructive override. Under monotonic constraints, health must classify exact constraint changes.

## Alcance

**In**:
1. Replace or supplement generic `field-override` with diagnostics such as valid narrowing, conflict, loosening, widening, destructive removal, and evolution-required.
2. Enforce required/severity/enum/type/domain/link/structural monotonic rules from T001.
3. Check duplicate domain conflicts across effective layers, not only within one parsed stem.
4. Update CLI validation tests for stem-health warnings/errors.

**Out**:
- Rewriting docs after final behavior; covered by T007.
- Fixing unrelated validation rules.

## Estado inicial esperado

- T002 provides layered resolver conflicts/provenance.

## Criterios de Aceptación

- Valid child narrowing is accepted or classified according to T001.
- Destructive child override fails under monotonic mode unless represented as explicit evolution.
- Stem-health diagnostics include actionable source paths and fields.
- `validate --all` surfaces monotonic diagnostics in batch output.

## Fuente de verdad

- `internal/rules/stemhealth.go`
- `internal/rules/stemhealth_test.go`
- `cmd/rootline/validate.go`
- `cmd/rootline/validate_test.go`
- `docs/validate.md`
