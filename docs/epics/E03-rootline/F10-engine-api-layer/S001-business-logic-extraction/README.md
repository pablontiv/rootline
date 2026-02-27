---
estado: Completed
tipo: historia
---
# S001: Business Logic Extraction

**Feature**: [F10 Engine API Layer](../README.md)
**Capacidad**: La lógica core de fix, migrate-split y doctor es invocable como library desde cualquier interfaz

## Antes / Despues

**Antes**: ~750 LOC de lógica de negocio vive en `cmd/rootline/` (fix.go, migrate.go, doctor.go). Solo accesible vía CLI. El futuro MCP server tendría que duplicar código o importar el paquete cmd.

**Despues**: CLI es capa delgada de orquestación (~40% menos LOC). Lógica core vive en `internal/fix`, `internal/migrate/split.go`, `internal/doctor`. Cualquier interfaz reutiliza el mismo engine.

## Criterios de Aceptacion (semanticos)

- [ ] `internal/fix` contiene toda la lógica de aplicación de propuestas y reescritura de frontmatter
- [ ] `internal/migrate/split.go` contiene los builders de YAML para split
- [ ] `internal/doctor` contiene todos los checks diagnósticos
- [ ] `cmd/rootline/fix.go`, `migrate.go` y `doctor.go` solo contienen orquestación CLI
- [ ] Todos los tests existentes siguen pasando

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-extract-fix-engine.md) | Extraer lógica de fix a internal/fix |
| [T002](T002-extract-migrate-split.md) | Mover split YAML builders a internal/migrate/split.go |
| [T003](T003-extract-doctor-engine.md) | Extraer checks diagnósticos a internal/doctor |

## Fuente de verdad

- `cmd/rootline/fix.go`
- `cmd/rootline/migrate.go`
- `cmd/rootline/doctor.go`
