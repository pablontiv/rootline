---
estado: Completed
---
# S002: Migrate rootline docs/epics .stem to levels

**Feature**: [F03 Schema Migration](../README.md)
**Capacidad**: rootline/docs/epics consolida ~70 child `.stem` files en un root `.stem` con `levels:`, manteniendo overrides donde necesario
**Cubre**: P5 del Epic — dogfooding con el proyecto rootline

## Antes / Despues

**Antes**: `docs/epics/.stem` (root) + child `.stem` en cada epic, feature, y story directory (~70 archivos). Cada child define `id: {type: sequence}` y opcionalmente `tipo`, `ejecutable_en`, `hold`. Contenido altamente repetitivo.

**Despues**: Un solo `docs/epics/.stem` con `levels:` declara la jerarquia completa. Child `.stem` files eliminados excepto los que tengan overrides genuinos (ej: `cliente` en F05). `rootline validate --all docs/epics/` produce los mismos resultados.

## Criterios de Aceptacion (semanticos)

- [ ] Root `.stem` reescrito con `levels:` section cubriendo epic, feature, story, task
- [ ] Child `.stem` files eliminados (excepto overrides genuinos)
- [ ] `rootline validate --all docs/epics/` pasa sin errores
- [ ] Effective schema per-level es identico al anterior

## Invariantes

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV5: Todos los documentos existentes siguen validando correctamente
  - Verificar: `rootline validate --all docs/epics/`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-consolidate-rootline-child-stems.md) | Consolidate rootline child .stem files into root levels and validate |

## Fuente de verdad

- `docs/epics/.stem` — current root stem
- `docs/epics/E03-rootline/.stem` — example child stem (feature level)
- `docs/epics/E03-rootline/F05-mcp-distribution/.stem` — example child stem (story level)
- `docs/epics/E03-rootline/F05-mcp-distribution/S001-mcp-server/.stem` — example child stem (task level)
