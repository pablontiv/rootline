---
estado: Specified
tipo: task
---
# T006: Extract schema generation services from init

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE2 y CE4 del Outcome.

[[blocked_by:./T005-normalize-proposal-taxonomy.md]]

## Preserva

- INV1: `.stem` mutation is explicit schema evolution, never a hidden side effect of data repair.
  - Verificar: extracted services return data/patches and do not write files.

## Contexto

Data-first bootstrap should reuse existing strengths: `init`, hierarchy inference, schema coverage, validation gaps, and migrate split. Today much of this is command-oriented and writes `.stem` files directly or prints YAML.

## Alcance

**In**:
1. Extract flat schema generation from `cmd/rootline/init.go` into reusable internal functions that return structured schema data or YAML patches without writing.
2. Extract hierarchical/match-based schema generation from `AnalyzeHierarchy`/init flow for reuse by `schema propose`.
3. Identify reusable pieces from `migrate --split` for proposed child `.stem` generation.
4. Add no-write unit tests for the extracted services.

**Out**:
- Adding `schema propose` CLI.
- Applying generated schema files.
- Changing inference heuristics unless needed to make generation reusable.

## Estado inicial esperado

- `rootline init` can already generate `.stem` from data, but its logic is coupled to CLI write behavior.

## Criterios de Aceptación

- Extracted service functions can produce flat and hierarchical schema candidates in tests without filesystem writes.
- The services preserve current `init --dry-run`/normal behavior when called by `init`.
- Bootstrap candidate generation includes fields, enum candidates, requiredness, sequences, sections, structural rules, and hierarchy metadata when available.

## Fuente de verdad

- `cmd/rootline/init.go`
- `internal/infer/infer.go`
- `internal/infer/hierarchy.go`
- `internal/infer/schema_coverage.go`
- `internal/infer/validation_gaps.go`
- `internal/migrate/split.go`
