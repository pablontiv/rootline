---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Agregar StructuralRules y SubdirRules structs a StemFile

**Story**: [S001 Structural Directory Rules](README.md)

## Contexto

Rootline define schemas en `.stem` files via `StemFile` struct (`internal/rules/rules.go`). Hoy solo soporta `schema:`, `validate:`, `links:`, `derive:` y `state:`. Se necesita un nuevo bloque `structural:` que permita definir reglas sobre la estructura de directorios (no sobre archivos individuales).

El bloque se ve asi en YAML:

```yaml
structural:
  subdirs:
    require_index: README.md
    min_children: 2
    max_children: 10
    severity: warn
```

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: StructuralRules
    metodos: []
  - nombre: SubdirRules
    metodos: []
dependencias_externas: []
tests:
  - StemFile con bloque structural se parsea correctamente desde YAML
  - StemFile sin bloque structural tiene Structural con valores zero
  - Merge de structural entre parent y child (child overrides)
```

## Dependencias

- Ninguna — primer task de la story

## Alcance

**In**:
1. Agregar structs `StructuralRules` y `SubdirRules` en `internal/rules/rules.go`
2. Agregar campo `Structural StructuralRules` a `StemFile`
3. Agregar merge logic para `Structural` en `internal/rules/merge.go`
4. Tests de parsing y merge

**Out**: No implementar la logica de validacion (eso es T002)

## Estado inicial esperado

- `internal/rules/rules.go` tiene `StemFile` struct sin campo `Structural`
- `internal/rules/merge.go` tiene `MergeStemFiles` sin logica para structural
- YAML parsing de .stem files ignora claves desconocidas (gopkg.in/yaml.v3 behavior)

## Criterios de Aceptacion

- `go test ./internal/rules/ -run TestStructural` pasa
- Un .stem con bloque `structural:` se parsea en `StemFile.Structural.Subdirs` con valores correctos
- Un .stem sin bloque `structural:` tiene `StemFile.Structural` con valores zero (no nil panic)
- Merge de parent con structural + child sin structural preserva parent values
- Merge de parent con structural + child con structural usa child values

## Fuente de verdad

- `internal/rules/rules.go` — StemFile struct (lineas 14-24)
- `internal/rules/merge.go` — MergeStemFiles function
- `internal/rules/rules_test.go` — tests existentes de parsing
