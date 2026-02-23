---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Clasificador de breaking changes

**Story**: [S001 Change Detection](README.md)

[[blocks:T001-migrate-dry-run]]

## Contexto

T001 establece la infraestructura de `rootline migrate`. Esta task implementa la logica de clasificacion que determina si un cambio de schema es breaking o no, y cuantos archivos afecta cada cambio.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/migrate
interfaces:
  - nombre: SchemaChange
    metodos: []
dependencias_externas: []
tests:
  - Field removed → breaking, lists files with that field
  - Enum value removed → breaking, lists files using that value
  - Type changed (string→enum) → breaking
  - Required false→true → breaking, lists files without field
  - Field added → non-breaking
  - Enum value added → non-breaking
  - Required true→false → non-breaking
```

## Dependencias

- T001 completado (migrate command y SchemaDiff infrastructure)
- internal/index/ (scanner para contar archivos afectados)

## Alcance

**In**:
1. `SchemaDiff(before, after StemFile) → DiffResult` con lista de SchemaChange
2. SchemaChange struct: Field, ChangeType, Breaking bool, Before, After, AffectedFiles int
3. ChangeType enum: field_added, field_removed, type_changed, enum_added, enum_removed, required_tightened, required_relaxed
4. Contar archivos afectados por cada breaking change (scan records)
5. Output sorting: breaking changes primero

**Out**: Auto-fix suggestions (eso es S002), severity levels, custom rules

## Estado inicial esperado

- T001 completado (migrate command structure)
- internal/index/ Scanner funcional

## Criterios de Aceptacion

- SchemaDiff produce lista correcta de SchemaChange para cada tipo de cambio
- Breaking flag es correcto para cada ChangeType
- AffectedFiles count es correcto (basado en scan de records)
- `go test ./internal/migrate/ -run TestClassifier -v` pasa

## Fuente de verdad

- `internal/migrate/` (nuevo package de T001)
- `internal/rules/rules.go` (StemFile.Schema field definitions)
- `internal/index/index.go` (Scanner para contar archivos)
