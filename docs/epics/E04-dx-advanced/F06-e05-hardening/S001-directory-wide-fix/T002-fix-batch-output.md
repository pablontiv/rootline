---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Batch JSON/table output para fix --all

**Story**: [S001 Directory-wide Fix](README.md)

## Contexto

Con T001, fix --all puede reparar directorios. Pero el output actual es lineal (una linea por archivo). Necesita un output batch estructurado igual que validate --all: JSON con summary y tabla con columnas.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: FixResult
    metodos:
      - nombre: struct fields
        input: "Path, Fixed bool, FieldsAdded int, ValuesCorrected int, Changes []string"
        output: "JSON serializable"
  - nombre: BatchFixResult
    metodos:
      - nombre: struct fields
        input: "Version, Kind, Results []FixResult, Summary{Total, Fixed, Skipped}"
        output: "JSON serializable"
dependencias_externas: []
tests:
  - fix --all JSON output tiene version 1, kind rootline/fix-batch, summary
  - fix --all table output tiene columnas File, Fixed, Changes
  - fix --all con 0 errores muestra summary con fixed=0
```

## Dependencias

- T001 (fix --all flag)
- BatchValidationResult pattern (`internal/rules/validate_result.go`)

## Alcance

**In**:
1. Crear structs `FixResult` y `BatchFixResult` en fix.go (o internal/rules/)
2. `BatchFixResult` con `Version: 1`, `Kind: "rootline/fix-batch"`, `Summary`
3. Modificar `runFixAll` para recolectar results y producir batch output
4. JSON output via `outputJSON(cmd, batch, hasErrors)`
5. Table output con `renderFixTable()`: columnas File, Fixed, Changes
6. Tests

**Out**: Per-file JSON output changes (solo batch), interactive mode

## Estado inicial esperado

- T001 completado (fix --all funcional con output lineal)
- `internal/rules/validate_result.go` con BatchValidationResult como referencia

## Criterios de Aceptacion

- `rootline fix --all --output json` retorna `{"version":1,"kind":"rootline/fix-batch","results":[...],"summary":{...}}`
- `rootline fix --all --output table` muestra tabla con File, Fixed, Changes
- Summary incluye total, fixed, skipped counts
- `go test ./... -race` pasa

## Fuente de verdad

- `cmd/rootline/fix.go` — archivo a modificar
- `internal/rules/validate_result.go` — BatchValidationResult pattern
- `cmd/rootline/validate.go:199-225` — renderValidateTable pattern
