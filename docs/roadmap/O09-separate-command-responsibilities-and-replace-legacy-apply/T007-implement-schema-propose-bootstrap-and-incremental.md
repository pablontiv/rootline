---
estado: Completed
tipo: task
---
# T007: Implement schema propose for bootstrap and incremental use

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE2 y CE4 del Outcome.

[[blocked_by:./T006-extract-schema-generation-services-from-init.md]]

## Preserva

- INV3: Existing read-only commands remain read-only.
  - Verificar: `schema propose` tests assert no files are created or modified.

## Contexto

Rootline needs a data-first mode for existing filesystem databases: discover candidate `.stem` constraints and emit a proposal, not mutate schema as part of `apply` or `fix`.

## Alcance

**In**:
1. Add a read-only `schema propose` command or equivalent command group.
2. Support bootstrap/no-stem, incremental/existing-stem, and governance signal modes.
3. Emit versioned JSON such as `rootline/schema-proposals` with explicit operation ids, target `.stem` paths, confidence, requires-agent flags, and patch previews.
4. Reuse analyze/init/hierarchy detectors and extracted schema generation services.
5. Add tests proving no filesystem writes.

**Out**:
- Applying schema proposals.
- Data repair proposal application.
- Monotonic `.stem` enforcement.

## Estado inicial esperado

- T005 has defined proposal taxonomy.
- T006 has extracted no-write schema generation services.

## Criterios de Aceptación

- `schema propose` emits parseable versioned JSON for bootstrap and incremental fixtures.
- Proposals include explicit target `.stem` paths and do not rely on `entries[0]` heuristics.
- Missing/implicit schema cases become proposals or diagnostics, not direct writes.
- `go test ./cmd/rootline ./internal/infer ./internal/migrate -run 'Schema|Propose|Init|Analyze'` passes focused cases.

## Fuente de verdad

- `cmd/rootline/analyze.go`
- `cmd/rootline/init.go`
- `internal/infer/report.go`
- `internal/infer/schema_coverage.go`
- `internal/infer/validation_gaps.go`
- `internal/infer/hierarchy.go`
