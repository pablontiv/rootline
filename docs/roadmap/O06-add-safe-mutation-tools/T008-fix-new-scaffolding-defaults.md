---
estado: Completed
tipo: task
---
# T008: Fix new scaffolding defaults

**Outcome**: [Add safe mutation tools](README.md)

## Preserva

- INV1: Mutating tools must not bypass Rootline validation or write outside user-approved paths.

## Contexto

Investigation confirmed rootline new --dry-run for a roadmap T*.md path emits tipo: outcome because generateMarkdown falls back to field.Values[0] when no explicit default exists. docs/new.md also claims sequence auto-generation that generateMarkdown does not implement.

## Alcance

**In**:
1. Define enum default policy for rootline new when no explicit default is present
2. Ensure roadmap T*.md scaffolds do not get tipo: outcome
3. Update docs/new.md to match actual enum and sequence behavior
4. Add tests for enum with explicit default, enum without default, and roadmap-style T*.md tipo

**Out**:
1. Full roadmapctl materialization changes unless required to consume the corrected Rootline behavior
2. Unrelated schema inference changes

## Estado inicial esperado

Pending

## Criterios de Aceptación

- rootline new --dry-run for a roadmap T*.md path does not emit tipo: outcome.
- Tests cover enum fields with explicit default and without explicit default.
- docs/new.md accurately describes enum scaffolding behavior.
- The discrepancy around type: sequence auto-generation is either fixed or documented with a test-backed limitation.

## Fuente de verdad

- cmd/rootline/new.go
- cmd/rootline/new_test.go
- docs/new.md
- docs/roadmap/.stem
