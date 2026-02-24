---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Integrate Aggregate Generator in Migrate Split

**Story**: [S002 Generator Integration](README.md)
**Contribuye a**: `rootline migrate --split` preserva aggregates existentes y genera faltantes

[[blocks:T001-implement-aggregate-generator]]

## Preserva

- INV1: Tests existentes siguen pasando
  - Verificar: `go test ./cmd/rootline/ -run TestMigrate -v`

## Contexto

`cmd/rootline/migrate.go` implementa `rootline migrate --split` que convierte un .stem flat en .stem jerárquicos per-level. Usa `infer.AnalyzeHierarchy()` para detectar niveles y `distributeFields()` para decidir qué campos van al root vs per-level. Actualmente NO genera `aggregate:` para campos enum del root, aunque sí copia aggregates existentes del .stem original.

## Especificacion Tecnica

**Modificar**: `cmd/rootline/migrate.go`
**Tests**: `cmd/rootline/migrate_test.go` — 2 tests nuevos

**Cambios en funciones existentes**:

1. `buildSplitStems()` (~línea 393):
   - Después de construir `rootFields` del split
   - Llamar `migrate.GenerateAggregates(rootFields, existing.Aggregate)` donde `existing.Aggregate` es el mapa de aggregates del .stem original
   - Esto preserva aggregates existentes y solo genera para campos enum sin aggregate

2. `buildSplitRootYAML()` (~línea 450):
   - Nuevo parámetro: `generatedAggregates map[string]string`
   - Después de emitir aggregates existentes (~líneas 537-548), emitir los generados
   - No duplicar: solo emitir generados que no estén en existentes

**2 tests nuevos en migrate_test.go**:
1. `TestMigrateSplit_GeneratesAggregate`: .stem flat con campo enum `estado` sin aggregate → split produce root .stem con `aggregate:` section
2. `TestMigrateSplit_PreservesExistingAggregate`: .stem flat con aggregate existente para `estado` → split preserva el existente, no genera duplicado

## Dependencias

> Requiere T001-implement-aggregate-generator (S001) completado.

- `internal/migrate` — `GenerateAggregates()` function

## Alcance

**In**:
1. Modificar `buildSplitStems()` para llamar al generador
2. Modificar `buildSplitRootYAML()` para emitir aggregate section generada
3. Agregar 2 tests en `migrate_test.go`

**Out**: No tocar `init.go`, `fix.go`, ni otros comandos.

## Estado inicial esperado

- `internal/migrate/aggregate.go` existe con `GenerateAggregates()` funcional (T001 de S001 completado)
- `cmd/rootline/migrate_test.go` tiene tests de split existentes como referencia

## Criterios de Aceptacion

- `go test ./cmd/rootline/ -run TestMigrateSplit -v` pasa (tests existentes + 2 nuevos)
- `rootline migrate --split` en directorio con .stem flat con enum sin aggregate → produce root .stem con `aggregate:`
- `rootline migrate --split` en directorio con .stem flat con aggregate existente → preserva el existente intacto

## Fuente de verdad

- `cmd/rootline/migrate.go` — buildSplitStems (~393), buildSplitRootYAML (~450, ~537-548)
- `cmd/rootline/migrate_test.go`
