---
estado: Specified
tipo: task
---
# T001: Design monotonic stem constraint algebra

**Outcome**: [O10 Move .stem to monotonic hierarchical constraints](README.md)
**Contribuye a**: CE1 del Outcome.

## Preserva

- INV1: Moving down the directory tree must never silently reduce parent guarantees.
  - Verificar: the design explicitly classifies reductions as invalid or explicit evolution.
- INV3: v2 compatibility or migration behavior is explicit.
  - Verificar: the design names the version/flag/migration strategy.

## Contexto

Research concluded that Rootline is closer to a hierarchical filesystem database than a config cascade. Parent `.stem` files should declare guarantees for descendants; child `.stem` files should add or narrow constraints. Current merge semantics allow destructive replacement, required loosening, array replacement, and nil removal.

## Alcance

**In**:
1. Define the partial order for field type, required, enum values, severity, match/scope, excludes, default, sequence, domain, section metadata, links, derive, aggregate, and structural rules.
2. Decide how child enum values behave: subset/narrowing, extension/widening, or explicit replacement.
3. Decide how defaults are classified: constraint, authoring hint, or generation policy.
4. Decide compatibility strategy: v3, `semantics: monotonic`, project setting, or migration command.
5. Document examples for valid narrowing and invalid destructive override.

**Out**:
- Implementing resolver changes.
- Applying schema evolution operations.

## Estado inicial esperado

- Current `MergeStemFiles` is a YAML cascade: maps merge, arrays/scalars replace, nil removes, schema fields replace wholesale with only severity tightening as an exception.

## Criterios de Aceptación

- A design document contains a matrix of allowed, rejected, and explicit-evolution operations.
- The design answers whether child `string -> enum` narrowing is valid.
- The design states how legacy v2 projects are handled.
- The design identifies tests needed for every constraint category.

## Fuente de verdad

- `internal/rules/merge.go`
- `internal/rules/rules.go`
- `internal/rules/merge_test.go`
- `internal/rules/stemhealth.go`
- `docs/levels.md`
- `README.md`
