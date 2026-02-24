---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Extract doctor checks to internal/doctor

**Story**: [S001 Business Logic Extraction](README.md)

## Contexto

`cmd/rootline/doctor.go` (321 LOC) contiene 7 checks diagnósticos (~165 LOC de lógica) mezclados con la orquestación CLI. Los checks son: yaml-valid, scope-match, type-consistency, enum-values, rule-field-exists, field-override, aggregated-required. Toda esta lógica es independiente de cobra y debe ser accesible para el MCP server.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/doctor
interfaces:
  - nombre: Doctor
    metodos:
      - nombre: RunChecks
        input: "absRoot string"
        output: "*Result, error"
dependencias_externas: []
tests:
  - RunChecks detecta YAML inválido en .stem
  - RunChecks detecta scope orphan (match sin archivos)
  - RunChecks detecta type inconsistency entre parent/child .stem
  - RunChecks detecta enum con menos de 2 valores
  - RunChecks detecta validate rule que referencia campo inexistente
  - RunChecks detecta field override de parent
  - RunChecks detecta required + aggregate en mismo campo
  - RunChecks retorna summary correcto (pass/warn/fail counts)
```

## Dependencias

- `internal/rules` — ParseStemFile, WalkUp, MergeStemFiles, StemFile

## Alcance

**In**:
1. Crear paquete `internal/doctor/`
2. Mover tipos: `DoctorResult` → `Result`, `DoctorCheck` → `Check`, `DoctorSummary` → `Summary`
3. Crear función `RunChecks(absRoot string) (*Result, error)` que encapsula toda la lógica de los 7 checks
4. Actualizar `cmd/rootline/doctor.go` → `runDoctor` delega a `doctor.RunChecks`, solo hace output formatting
5. Crear `internal/doctor/doctor_test.go` con tests unitarios usando `t.TempDir()`
6. Verificar tests existentes en `cmd/rootline/doctor_test.go`

**Out**: No agregar nuevos checks. No cambiar la lógica interna de los checks existentes. No modificar el formato de output JSON/table.

## Estado inicial esperado

- `cmd/rootline/doctor.go` contiene toda la lógica incluyendo tipos y checks
- `internal/doctor/` no existe
- `cmd/rootline/doctor_test.go` pasa

## Criterios de Aceptacion

- `go test ./internal/doctor/ -race` pasa con >= 80% coverage
- `go test ./cmd/rootline/ -run TestDoctor -race` pasa
- `go test ./... -race` pasa
- `cmd/rootline/doctor.go` no contiene lógica de checks (solo routing y output)
- `internal/doctor/` exporta `RunChecks`, `Result`, `Check`, `Summary`

## Fuente de verdad

- `cmd/rootline/doctor.go` — funciones y tipos a extraer
- `cmd/rootline/doctor_test.go` — tests existentes
