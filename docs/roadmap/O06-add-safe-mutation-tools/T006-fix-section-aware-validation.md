---
estado: Specified
tipo: task
---
# T006: Fix section-aware validation

**Outcome**: [Add safe mutation tools](README.md)

## Preserva

- INV1: Mutating tools must not bypass Rootline validation or write outside user-approved paths.

## Contexto

Investigation showed rootline set parses AST and can create a required section, but rootline validate uses the non-AST registry and then reports the created section as missing because record.Sections is empty.

## Alcance

**In**:
1. Audit single-file validate and validate --all extraction paths for section-aware validation
2. Ensure required type: section fields are validated from parsed sections
3. Add regression tests for required section present, missing, and created by rootline set --create

**Out**:
1. Changing section schema syntax beyond what is needed for validation correctness
2. Implementing ordered section validation unless required by an existing contract

## Estado inicial esperado

Pending

## Criterios de Aceptación

- After rootline set --create <file> investigacion=... creates a required section, rootline validate <file> exits 0 for that section requirement.
- rootline validate --all also detects required sections correctly when .stem contains type: section fields.
- Tests cover required section present, absent, and created through the set pipeline.
- Existing frontmatter validation tests continue to pass.

## Fuente de verdad

- cmd/rootline/validate.go
- cmd/rootline/set.go
- internal/rules/validate.go
- internal/extract/registry.go
- internal/e2e/set_test.go
- cmd/rootline/set_test.go
