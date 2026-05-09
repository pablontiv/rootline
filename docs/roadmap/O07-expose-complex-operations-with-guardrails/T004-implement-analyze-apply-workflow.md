---
estado: Completed
tipo: task
---
# T004: Implement protected workflow around analyze reports and apply.

**Outcome**: [O07 Expose complex operations with guardrails](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T001-design-complex-operation-ux.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T007-implement-schema-propose-bootstrap-and-incremental.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T008-implement-schema-apply-explicit.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T009-implement-repair-apply-data-only.md]]
[[blocked_by:../O09-separate-command-responsibilities-and-replace-legacy-apply/T011-deprecate-legacy-apply-and-update-command-surfaces.md]]

## Preserva

- INV1: Bulk operations require explicit user intent and cannot be triggered silently by agent context guidance.
  - Verificar: Inspect commands, prompts, and tool activation defaults.

## Contexto

Esta task forma parte de O07 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Implement protected workflow around analyze reports and apply.

## Alcance

**In**:
1. Analyze report can be generated and inspected from Pi.
2. Apply requires explicit user approval, target report, and post-apply validation.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-design-complex-operation-ux.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Analyze report can be generated and inspected from Pi.
- Apply requires explicit user approval, target report, and post-apply validation.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `internal/infer/`
- `cmd/rootline/analyze.go`
- `cmd/rootline/apply.go`

## Salida / Implementación

### Tools Implemented

1. **rootline-analyze** (read-only)
   - Input: `path` (required), `incremental` (optional), `threshold` (optional)
   - Runs: `rootline analyze <path> --output json [--incremental] [--threshold N]`
   - Timeout: 60s
   - Returns: Full AnalyzeReport JSON
   - No confirmation required (read-only operation)

2. **rootline-apply** (mutating)
   - Input: `report_path` (required), `mode` ("schema" | "repair" | "both"), `dry_run` (optional, default: true), `confirmed` (optional)
   - Validates `report_path` with `validateTargetPath()`
   - Requires confirmation when `!dry_run` in non-interactive mode
   - Runs post-apply validation with `rootline validate --all`
   - Returns: `{ applied, dry_run, mode, schema_result?, repair_result?, validation? }`
   - Timeout: 30s per apply command

### Implementation Details

- File: `integrations/pi/extensions/analyze-apply.ts`
- Test file: `integrations/pi/extensions/analyze-apply.test.ts`
- Test suite: 19 tests covering all paths (analyze success, apply dry-run, confirmation, validation, error handling)
- Security: Path validation prevents traversal attacks
- Guardrails: Confirmation required for non-dry-run mutations in non-interactive mode

### Acceptance Criteria Met

- ✓ Analyze report can be generated and inspected from Pi
- ✓ Apply requires explicit user approval, target report, and post-apply validation
- ✓ `rootline validate --all docs/roadmap/` returns exit 0 (79 valid files, 2 schema-health warnings only)
