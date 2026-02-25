---
estado: Pending
tipo: feature
---
# F11: Aggregate Auto-Generation & Command Consolidation

**Epic**: [E03](../README.md)
**Satisface**: Rootline binario funcional (métrica de éxito E03)
**Objetivo**: rootline detecta campos enum jerárquicos sin aggregate y genera/aplica expresiones automáticamente; validate absorbe doctor checks
**Beneficio**: Elimina drift silencioso de estado en jerarquías y reduce 3 comandos de health-check a 2 (validate + fix)
**Milestone**: `rootline fix --all --dry-run` sobre stem sin aggregate reporta AddAggregate proposals AND `rootline validate` reporta stem health checks AND `rootline doctor` emite deprecation warning

## Scope

**In**: Generador de aggregate, integración en init/migrate/fix, unificación validate+doctor
**Out**: Cambios al motor de agregación existente (`internal/derive/`), nuevos tipos de proposal más allá de AddAggregate

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-aggregate-expression-generator/) | Aggregate Expression Generator | Librería que genera expresiones aggregate desde valores enum |
| [S002](S002-generator-integration/) | Generator Integration | init y migrate auto-generan aggregate para enums jerárquicos |
| [S003](S003-fix-proposal-missing-aggregates/) | Fix Proposal Missing Aggregates | fix detecta y aplica aggregate faltantes en stems existentes |
| [S004](S004-validate-doctor-unification/) | Validate & Doctor Unification | validate absorbe doctor checks, doctor queda como alias deprecado |

## Invariantes

- INV1 (heredado E03): Tests existentes siguen pasando (`go test ./... -race`)
- INV2: Coverage >= 85%

## Dependencias

- Ninguna — trabajo independiente de F05/F09/F10

## Fuente de verdad

- `internal/migrate/` — aggregate generator
- `internal/proposal/proposal.go` — proposal engine
- `cmd/rootline/init.go`, `cmd/rootline/migrate.go` — CLI integration
- `cmd/rootline/validate.go`, `cmd/rootline/doctor.go` — command unification
