---
estado: Pending
tipo: historia
cliente: Platform Owner
---
# S002: Dependency State Propagation

**Feature**: [F01 Link Field Resolution](../README.md)
**Capacidad**: .stem derive expressions calculan estado (blocked/unblocked) basado en dependencias via wiki-links

## Antes / Despues

**Antes**: Un task con `[[blocks:T001]]` no tiene forma de saber si T001 esta Completado. El developer debe verificar manualmente. `rootline query --where "estado == 'Pending'"` retorna tasks que en realidad estan bloqueadas.

**Despues**: El .stem tiene `derive.estado` que usa `blocked_by` (inyectado por S001). Si todos los blockers estan Completados, el task se deriva como Pending. Si alguno no esta Completado, se deriva como Bloqueada. `rootline query --where "estado == 'Pending'"` retorna solo tasks genuinamente accionables.

## Criterios de Aceptacion (semanticos)

- [ ] .stem derive expression usa blocked_by para calcular estado
- [ ] Tasks con blockers Completados se derivan como Pending
- [ ] Tasks con blockers no-Completados se derivan como Bloqueada
- [ ] E2E tests verifican dependency chains reales

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-blocked-unblocked-derivation.md) | Configurar derive expression para blocked/unblocked en .stem |
| [T002](T002-e2e-dependency-tests.md) | Tests e2e con dependency chains reales |

## Fuente de verdad

- `docs/epics/.stem` (aggregate y derive expressions)
- `internal/e2e/` (test fixtures)
