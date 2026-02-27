---
estado: Specified
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T002: Integrate drift warnings in validation output

**Story**: [S001 Parent-child drift detection](README.md)
**Contribuye a**: Drift warnings appear in both table and JSON validation output

[[blocks:T001-implement-drift-comparison]]

## Preserva

- INV1: All existing workflows produce identical results for v1 stems
  - Verificar: `go test ./... -race` passes
- INV4: Drift warnings are warnings, not errors
  - Verificar: Validation exit code is 0 when only drift warnings exist

## Contexto

With `DetectDrift` implemented (T001), this task integrates drift detection into the validation pipeline. When `rootline validate` runs on a directory, it should check index files against their children and include drift warnings in the output. Drift warnings must be distinct from validation errors — they use a different severity level and don't cause validation failure.

## Especificacion Tecnica

Modify validation pipeline:
- Add `DriftWarnings []DriftWarning` to `ValidationResult` (or `BatchValidationResult`)
- In the batch validation flow (`validate --all` or directory validation), after validating individual records, call `DetectDrift` for each index file and its direct children
- Add drift warnings to output:
  - Table format: new section "Drift Warnings" with field, parent path, parent value, children value
  - JSON format: `"drift_warnings"` array in the result
- Drift warnings don't increment error count or affect exit code

## Dependencias

- T001: DetectDrift function must exist

## Alcance

**In**:
1. Add DriftWarnings to ValidationResult
2. Call DetectDrift during batch validation
3. Display drift warnings in table and JSON output
4. Ensure exit code 0 when only drift warnings exist
5. Unit tests for integration

**Out**: Match-scoped exclusion (T003), E2E tests (S002)

## Estado inicial esperado

- T001 completed: `DetectDrift` function exists
- `internal/rules/validate.go` has `ValidationResult` and batch validation logic

## Criterios de Aceptacion

- `rootline validate` on a directory with drift shows drift warnings in output
- JSON output includes `"drift_warnings"` array
- Exit code is 0 when only drift warnings exist (no actual errors)
- `go test ./internal/rules/ -run TestValidateWithDrift` passes

## Fuente de verdad

- `internal/rules/validate.go` — ValidationResult, batch validation
- `cmd/rootline/validate.go` — CLI output formatting
