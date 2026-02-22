---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Create Proposal Types and Basic Detectors

**Story**: [S002 Proposal Analysis Engine](README.md)

## Contexto

El motor de propuestas necesita tipos base y la logica de analisis central. Este task crea el package `internal/proposal/` con los tipos `Proposal`, `Report`, `Summary`, la funcion `Analyze()` orquestadora, y los 3 detectores mas simples: `extend_enum` (valor invalido compartido por N>=2 archivos → proponer agregar al .stem), `correct_value` (Levenshtein closest — migrar de fix.go), `add_field` (campo requerido faltante → default).

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/proposal
interfaces:
  - nombre: Proposal
    metodos: []
  - nombre: Report
    metodos: []
  - nombre: Analyze
    metodos:
      - nombre: Analyze
        input: "records []*extract.Record, effective *rules.StemFile, errs map[string][]rules.ValidationError"
        output: "*Report"
dependencias_externas: []
tests:
  - 3 records con "Obsoleto" enum error → 1 extend_enum proposal
  - 1 record con "Completo" enum error → 1 correct_value proposal (Levenshtein → "Completado")
  - 1 record sin campo required → 1 add_field proposal
  - 0 errores → Report con 0 proposals
```

## Alcance

**In**:
1. Crear `internal/proposal/proposal.go` con tipos: `Type` (string constants), `Proposal`, `Report`, `Summary`
2. Crear `internal/proposal/detect.go` con: `detectExtendEnum()`, `detectCorrectValue()`, `detectAddField()`
3. Implementar `Analyze()` que ejecuta detectores en orden de prioridad y retorna Report
4. Migrar `closestMatch()` y `levenshtein()` desde `cmd/rootline/fix.go` a este package (o importar)
5. Tests en `internal/proposal/proposal_test.go`

**Out**: No modificar fix.go todavia — eso es S003. No implementar migrate_value, extract_body, infer_from_children (T002 y T003).

## Estado inicial esperado

- `internal/proposal/` no existe
- `internal/rules/validate.go` tiene `ValidationError` type
- `internal/extract/extract.go` tiene `Record` type
- `cmd/rootline/fix.go` tiene `closestMatch()` y `levenshtein()` como referencia

## Criterios de Aceptacion

- `go test ./internal/proposal/ -run TestDetectExtendEnum -v` pasa: 3 records con mismo valor invalido → 1 proposal tipo extend_enum
- `go test ./internal/proposal/ -run TestDetectCorrectValue -v` pasa: 1 record con typo → closest match
- `go test ./internal/proposal/ -run TestDetectAddField -v` pasa: record sin required field → add_field proposal
- `go test ./internal/proposal/ -run TestAnalyze -v` pasa: Report con proposals priorizados
- `go vet ./internal/proposal/` sin errores

## Fuente de verdad

- `internal/proposal/` — package nuevo
- `cmd/rootline/fix.go:300-348` — closestMatch/levenshtein a reusar
- `internal/rules/validate.go` — ValidationError type
- `internal/rules/rules.go` — StemFile, SchemaField types
