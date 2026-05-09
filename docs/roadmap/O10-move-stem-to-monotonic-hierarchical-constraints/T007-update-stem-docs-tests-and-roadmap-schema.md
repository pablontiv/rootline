---
estado: Specified
tipo: task
---
# T007: Update stem docs tests and roadmap schema

**Outcome**: [O10 Move .stem to monotonic hierarchical constraints](README.md)
**Contribuye a**: CE5 del Outcome.

[[blocked_by:./T003-expose-stem-provenance-in-describe-explain.md]]
[[blocked_by:./T004-upgrade-stem-health-monotonic-diagnostics.md]]
[[blocked_by:./T005-adapt-validation-and-query-flows-to-layered-resolver.md]]
[[blocked_by:./T006-add-schema-evolution-for-destructive-changes.md]]

## Preserva

- INV3: v2 compatibility or migration behavior is explicit.
  - Verificar: docs and tests state legacy versus monotonic semantics.

## Contexto

Docs and tests currently encode cascade override behavior. The roadmap schema itself also shows a design tension: parent `docs/.stem` defines `estado: string`, while the canonical roadmap skill expects a status enum under `docs/roadmap/.stem`. Under monotonic semantics this should be valid narrowing or explicitly modeled.

## Alcance

**In**:
1. Update README and `.stem` docs from override cascade language to layered constraints where applicable.
2. Update tests that currently expect destructive child override or nil removal as normal behavior.
3. Resolve roadmap schema drift between `docs/roadmap/.stem` and `.claude/skills/roadmap/base.stem` according to approved narrowing semantics.
4. Update validate/describe docs for new health diagnostics and provenance output.
5. Fix stale docs discovered during investigation where touched by `.stem` semantics.

**Out**:
- New command surfaces from O09 unless needed for docs cross-links.
- Pi package implementation.

## Estado inicial esperado

- T003-T006 have implemented behavior and diagnostics.

## Criterios de Aceptación

- Docs no longer imply unsupported or unsafe `.stem` override behavior.
- Tests clearly distinguish legacy v2 and monotonic semantics.
- Roadmap schema status handling is consistent with configured status values and stem-health policy.
- `go test ./...` passes.
- `rootline validate --all docs/roadmap/` passes with no unexpected warnings.

## Fuente de verdad

- `README.md`
- `docs/levels.md`
- `docs/describe.md`
- `docs/validate.md`
- `internal/rules/merge_test.go`
- `internal/rules/stemhealth_test.go`
- `docs/.stem`
- `docs/roadmap/.stem`
- `.claude/skills/roadmap/base.stem`
