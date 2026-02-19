---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Definir JSON output contract para validation results

**Story**: [S001 Validation Engine](README.md)

## Contexto

Todos los outputs de Rootline siguen versioned JSON contracts. El output de validacion debe ser consumible por CI pipelines, hooks de Claude Code, y humanos. El contrato incluye version, kind, path del archivo validado, y lista de errores tipados.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: ValidationResult
    metodos:
      - nombre: ToJSON
        input: (none)
        output: "[]byte, error"
dependencias_externas: []
tests:
  - Documento con errores produce JSON con version, kind, errors
  - Documento valido produce JSON con errors vacio
  - JSON output es parseable y estable
  - Batch validation (multiples archivos) produce array de results
```

## Dependencias

- T001 completado (ValidationError struct disponible)

## Alcance

**In**:
1. Struct `ValidationResult` con Version (int), Kind ("rootline/validate"), Path, Errors []ValidationError, Valid (bool)
2. Para batch: `BatchValidationResult` con Version, Kind ("rootline/validate-batch"), Results []ValidationResult, Summary (total, valid, invalid counts)
3. JSON serialization con `encoding/json`
4. Contract version: 1

**Out**: CLI command wiring (S002), table output format

## Estado inicial esperado

- ValidationError struct de T001 disponible
- `encoding/json` stdlib

## Criterios de Aceptacion

- Single file result serializa a: `{"version":1,"kind":"rootline/validate","path":"...","valid":false,"errors":[...]}`
- Batch result serializa a: `{"version":1,"kind":"rootline/validate-batch","results":[...],"summary":{...}}`
- `Valid` es true cuando errors es vacio
- JSON output es stable (fields en orden consistente)

## Fuente de verdad

- `src/rootline/README.md` seccion "Versioning" (contract versioning)
- `src/rootline/docs/intent/v0-rootline.md` seccion 4 (Stable JSON Contracts)
