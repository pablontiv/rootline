---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Integrate nesting check in validate and fix pipelines

**Story**: [S002 Nesting Validation](README.md)
**Contribuye a**: Nesting errors se integran en validate y fix pipelines

[[blocks:T001-implement-checknesting-function]]

## Preserva

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV3: `.stem` sin `levels:` zero regression
  - Verificar: existing validate tests pass unchanged

## Contexto

`CheckNesting` exists as a standalone function. It needs to be called as part of the validate pipeline so that `rootline validate` reports nesting violations alongside schema errors. The fix pipeline should also be aware of nesting errors (though nesting errors are not auto-fixable — they indicate structural problems).

In `cmd/rootline/validate.go`, the validate pipeline (single file at line ~94 and batch at line ~166) calls WalkUp+Merge then Validate. After this task, it should also call CheckNesting when the effective stem has Levels.

Nesting errors should appear in the same ValidationResult output alongside schema errors, with a distinct rule name (e.g., "nesting").

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: validate pipeline (extension)
    metodos:
      - nombre: runValidate / runValidateAll
        input: existing args
        output: existing output + nesting errors
dependencias_externas: []
tests:
  - Existing validate tests pass unchanged
  - New test with levels fixture shows nesting error in output
```

## Dependencias

- S002/T001: CheckNesting function must exist

## Alcance

**In**:
1. In `cmd/rootline/validate.go`, after schema validation, call `CheckNesting` when stem has Levels
2. Append nesting errors to ValidationResult.Errors
3. Ensure JSON output includes nesting errors with rule "nesting"
4. Update fix pipeline to surface nesting errors (informational, not auto-fixable)
5. Add test verifying nesting errors appear in validate output

**Out**: MCP tools integration (F02), stemhealth (F02)

## Estado inicial esperado

- `CheckNesting` function exists in `internal/rules/hierarchy.go`
- `cmd/rootline/validate.go` has single file + batch validate pipelines
- ValidationResult struct has Errors field

## Criterios de Aceptacion

- `rootline validate` on a file with nesting violation reports error with rule "nesting"
- `rootline validate` on correctly nested files reports no nesting errors
- `rootline validate` on files without `levels:` in stem reports no nesting errors
- `go test ./... -race` passes
- JSON output includes nesting errors in `errors` array

## Fuente de verdad

- `cmd/rootline/validate.go` — validate pipeline (lines 90-190)
- `internal/rules/hierarchy.go` — CheckNesting
- `internal/rules/validate.go` — ValidationResult type
