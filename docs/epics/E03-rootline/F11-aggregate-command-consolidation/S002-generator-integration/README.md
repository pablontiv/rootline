---
estado: Pending
tipo: historia
---
# S002: Generator Integration in Init & Migrate

**Feature**: [F11 Aggregate Auto-Generation & Command Consolidation](../README.md)
**Capacidad**: `rootline init` y `rootline migrate --split` auto-generan expresiones aggregate para campos enum jerárquicos
**Cubre**: Integración del generador en los dos comandos que producen .stem files

## Antes / Despues

**Antes**: `rootline init` y `migrate --split` generan .stem sin sección `aggregate:`, produciendo jerarquías donde el estado de padres no se calcula automáticamente de hijos.

**Despues**: Ambos comandos detectan campos enum compartidos entre niveles y emiten sección `aggregate:` con expresiones generadas + note a stderr informando al usuario.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline init` en jerarquía con enum estado produce .stem con aggregate
- [ ] `rootline migrate --split` preserva aggregates existentes y genera para campos faltantes
- [ ] Ambos comandos emiten note a stderr cuando generan aggregate

## Invariantes

- INV1: Tests existentes siguen pasando
  - Verificar: `go test ./cmd/rootline/ -run "TestInit|TestMigrate" -v`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-integrate-aggregate-init.md) | Integrar generador en init.go para hierarchical init |
| [T002](T002-integrate-aggregate-migrate.md) | Integrar generador en migrate.go para split |

## Fuente de verdad

- `cmd/rootline/init.go` — buildHierarchicalStems, generateHierarchicalRootYAML
- `cmd/rootline/migrate.go` — buildSplitStems, buildSplitRootYAML
