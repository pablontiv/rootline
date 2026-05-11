---
estado: Specified
tipo: task
---
# T007: Reconcile set create contract

**Outcome**: [Add safe mutation tools](README.md)

## Preserva

- INV1: Mutating tools must not bypass Rootline validation or write outside user-approved paths.

## Contexto

Investigation confirmed rootline set --create currently fails before mutation for nonexistent files because set reads the target file before --create handling. The help text says it creates sections, while docs/set.md says it creates/scaffolds missing files.

## Alcance

**In**:
1. Decide and codify the supported --create behavior
2. Align CLI long help and docs/set.md with implemented behavior
3. Add tests for nonexistent target with --create and existing file missing section with --create
4. Clarify --no-validate behavior relative to enum pre-validation

**Out**:
1. Broad redesign of rootline new or materialize flows
2. Bulk creation of roadmap files outside the chosen set contract

## Estado inicial esperado

Pending

## Criterios de Aceptación

- The command help, docs/set.md, and tests describe the same --create behavior.
- A test covers rootline set --create on a nonexistent file and asserts the chosen result.
- A test covers rootline set --create creating a missing section in an existing file.
- The docs accurately state whether --no-validate bypasses only post-validation or also pre-validation.

## Fuente de verdad

- cmd/rootline/set.go
- cmd/rootline/set_test.go
- internal/fix/fix.go
- docs/set.md
