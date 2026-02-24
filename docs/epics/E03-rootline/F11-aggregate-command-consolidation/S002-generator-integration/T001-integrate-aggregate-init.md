---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Integrate Aggregate Generator in Init

**Story**: [S002 Generator Integration](README.md)
**Contribuye a**: `rootline init` en jerarquía con enum estado produce .stem con aggregate

[[blocks:T001-implement-aggregate-generator]]

## Preserva

- INV1: Tests existentes siguen pasando
  - Verificar: `go test ./cmd/rootline/ -run TestInit -v`

## Contexto

`cmd/rootline/init.go` implementa `rootline init` para bootstrappear .stem files. En modo jerárquico (`runInitHierarchical`), detecta niveles de directorio (E##/F##/S###/T###) y genera un .stem root + .stem per-level. Actualmente NO genera sección `aggregate:` para campos enum compartidos entre niveles, lo que causa que READMEs padre mantengan `estado` manual que diverge de los hijos.

## Especificacion Tecnica

**Modificar**: `cmd/rootline/init.go`
**Tests**: `cmd/rootline/init_test.go` — 2 tests nuevos

**Cambios en funciones existentes**:

1. `buildHierarchicalStems()` (~línea 153):
   - Después de construir `rootFields` (mapa de campos del schema root)
   - Llamar `migrate.GenerateAggregates(rootFields, nil)` para obtener expresiones
   - Retornar las expresiones generadas junto con el resultado actual

2. `generateHierarchicalRootYAML()` (~línea 184):
   - Nuevo parámetro: `generatedAggregates map[string]string`
   - Después de emitir sección `schema:`, emitir sección `aggregate:` con cada expresión generada
   - Usar formato multiline YAML (`|`) para las expresiones

3. `runInitHierarchical()` (~línea 107):
   - Para cada aggregate generado, emitir note a stderr: `"Note: auto-generated aggregate for '%s'"`

**2 tests nuevos en init_test.go**:
1. `TestInitAutoHierarchy_GeneratesAggregate`: directorio con 2 niveles y campo enum `estado` → .stem root contiene `aggregate:` section
2. `TestInitAutoHierarchy_NoAggregateForNonEnum`: directorio con campos string → no genera aggregate

## Dependencias

> Requiere T001-implement-aggregate-generator (S001) completado.

- `internal/migrate` — `GenerateAggregates()` function

## Alcance

**In**:
1. Modificar `buildHierarchicalStems()` para llamar al generador
2. Modificar `generateHierarchicalRootYAML()` para emitir aggregate section
3. Modificar `runInitHierarchical()` para emitir note a stderr
4. Agregar 2 tests en `init_test.go`

**Out**: No tocar `migrate.go`, `fix.go`, ni otros comandos.

## Estado inicial esperado

- `internal/migrate/aggregate.go` existe con `GenerateAggregates()` funcional (T001 de S001 completado)
- `cmd/rootline/init_test.go` tiene `TestInitAutoHierarchy` existente como referencia de pattern

## Criterios de Aceptacion

- `go test ./cmd/rootline/ -run TestInitAutoHierarchy -v` pasa (tests existentes + 2 nuevos)
- `rootline init` en directorio temporal con 2 niveles y campo enum `estado` produce .stem con `aggregate:` section
- Note a stderr visible: `"Note: auto-generated aggregate for 'estado'"`

## Fuente de verdad

- `cmd/rootline/init.go` — buildHierarchicalStems (~153), generateHierarchicalRootYAML (~184), runInitHierarchical (~107)
- `cmd/rootline/init_test.go` — TestInitAutoHierarchy (~144)
