---
estado: In Progress
tipo: task
---
# T008: Implement explicit schema apply

**Outcome**: [O09 Separate command responsibilities and replace legacy apply](README.md)
**Contribuye a**: CE4 del Outcome.

[[blocked_by:./T007-implement-schema-propose-bootstrap-and-incremental.md]]

## Preserva

- INV1: `.stem` mutation is explicit schema evolution, never a hidden side effect of data repair.
  - Verificar: command rejects non-schema proposal kinds and never modifies Markdown.
- INV2: Every machine-readable command used by agents emits parseable, versioned JSON.
  - Verificar: tests parse schema apply stdout.

## Contexto

Schema mutation should happen only through a command that accepts schema proposals with explicit `.stem` targets. This replaces the schema side of legacy `apply` and avoids ambiguous report-wide target selection.

## Alcance

**In**:
1. Add explicit schema apply command/engine that accepts only schema proposal reports.
2. Implement true dry-run for create/update/split `.stem` operations.
3. Apply only operation types with explicit target paths and supported semantics.
4. Validate changed `.stem` files and affected documents after apply.
5. Reject unsupported or agent-required operations unless an explicit approved mode exists.

**Out**:
- Data repair application.
- Full rollback if product decision says preflight-only is enough; document chosen behavior.
- Monotonic v3 enforcement beyond current semantics.

## Estado inicial esperado

- T007 produces schema proposal reports.

## Criterios de Aceptación

- `schema apply --dry-run` leaves `.stem` and Markdown bytes unchanged.
- Non-dry-run schema apply modifies only declared `.stem` targets.
- Command rejects wrong `kind/version`, ambiguous targets, and unsupported operations with structured errors.
- Post-apply validation runs or reports explicit validation commands and results.
- Focused Go tests and `rootline validate --all docs/roadmap/` pass.

## Fuente de verdad

- `internal/infer/apply.go`
- `internal/infer/scaffold.go`
- `internal/migrate/split.go`
- `cmd/rootline/apply.go`
- `cmd/rootline/validate.go`
