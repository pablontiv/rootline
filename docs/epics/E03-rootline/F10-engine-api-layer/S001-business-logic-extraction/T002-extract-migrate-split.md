---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Move split YAML builders to internal/migrate/split.go

**Story**: [S001 Business Logic Extraction](README.md)

## Contexto

`cmd/rootline/migrate.go` contiene 3 funciones (~222 LOC) que generan YAML para la operación `migrate --split`: `buildSplitStems`, `buildSplitRootYAML`, y `buildSplitChildYAML`. Estas funciones son lógica pura de transformación de datos (StemFile + HierarchyResult → YAML strings) sin dependencia de cobra. El paquete `internal/migrate/` ya existe con rename, diff y source loading — solo falta agregar split.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/migrate
interfaces:
  - nombre: SplitStems
    metodos:
      - nombre: BuildSplitStems
        input: "absTarget string, existing *rules.StemFile, hierarchy *infer.HierarchyResult"
        output: "[]StemOutput"
dependencias_externas:
  - gopkg.in/yaml.v3
tests:
  - BuildSplitStems distribuye campos root vs per-level correctamente
  - BuildSplitRootYAML preserva derive, aggregate, links, structural, validate
  - BuildSplitChildYAML genera sequence id y campos específicos del nivel
  - Campos con severity se preservan en el output
```

## Dependencias

- `internal/rules` — tipos StemFile, SchemaField
- `internal/infer` — tipos HierarchyResult, LevelSchema

## Alcance

**In**:
1. Crear `internal/migrate/split.go`
2. Mover funciones: `buildSplitStems`, `buildSplitRootYAML`, `buildSplitChildYAML`
3. Mover tipo local `stemFile` como `StemOutput` (exportado) con campos `Path string` y `Content string`
4. Exportar: `BuildSplitStems` (las otras quedan como helpers internos)
5. Actualizar `cmd/rootline/migrate.go` → `runMigrateSplit` delega a `migrate.BuildSplitStems`
6. Crear `internal/migrate/split_test.go`
7. Verificar que `cmd/rootline/migrate_test.go` sigue pasando

**Out**: No refactorizar `runMigrateSplit` más allá de la delegación. No tocar rename, diff ni source loading.

## Estado inicial esperado

- `cmd/rootline/migrate.go` contiene `buildSplitStems` (línea 393), `buildSplitRootYAML` (línea 449), `buildSplitChildYAML` (línea 576)
- `internal/migrate/` existe con `rename.go`, `diff.go`, `source.go`
- `cmd/rootline/migrate_test.go` pasa

## Criterios de Aceptacion

- `go test ./internal/migrate/ -race` pasa incluyendo nuevos tests de split
- `go test ./cmd/rootline/ -run TestMigrate -race` pasa
- `go test ./... -race` pasa
- `cmd/rootline/migrate.go` no contiene `buildSplitStems`, `buildSplitRootYAML`, `buildSplitChildYAML`
- `internal/migrate/split.go` exporta `BuildSplitStems` y tipo `StemOutput`

## Fuente de verdad

- `cmd/rootline/migrate.go` — funciones a extraer (líneas 393-616)
- `cmd/rootline/migrate_test.go` — tests existentes
- `internal/migrate/` — paquete destino
