---
estado: Completed
tipo: feature
---
# F10: Engine API Layer

**Epic**: [E03](../README.md)
**Objetivo**: La lógica core de rootline es invocable desde cualquier interfaz (CLI, MCP, library) sin duplicación
**Beneficio**: Desbloquea el MCP server (F05/S001) y elimina ~750 LOC de lógica de negocio atrapada en cmd/rootline/
**Milestone**: `internal/fix`, `internal/doctor` y `internal/migrate/split.go` existen con tests, y todas las funciones core aceptan `context.Context`

## Scope

**In**: Extracción de lógica de fix.go, migrate.go (split), doctor.go a paquetes internal/. Propagación de context.Context en interfaces core.
**Out**: Implementación del MCP server (eso es F05/S001). Paralelización. Refactor de validate.go o query.go (ya están bien separados).

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-business-logic-extraction/) | Business Logic Extraction | Lógica de fix, migrate-split y doctor accesible como library |
| [S002](S002-context-aware-engine/) | Context-Aware Engine | Todas las operaciones core soportan cancelación y timeouts |

## Dependencias

- Ninguna — este Feature habilita F05/S001 (MCP server)

## Fuente de verdad

- `cmd/rootline/fix.go` — 731 LOC, ~365 extractables
- `cmd/rootline/migrate.go` — 616 LOC, ~222 extractables (split builders)
- `cmd/rootline/doctor.go` — 321 LOC, ~165 extractables
- `internal/fix/` — no existe aún
- `internal/doctor/` — no existe aún
