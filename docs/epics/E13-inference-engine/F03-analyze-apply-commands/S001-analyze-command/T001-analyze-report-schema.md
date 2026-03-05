---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Definir analyze report JSON schema

**Story**: [S001 Analyze Command & Report Format](README.md)
**Contribuye a**: Report JSON schema definido con version: 1

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV2: Contratos JSON mantienen `"version": 1`
  - Verificar: Struct tiene `Version int json:"version"` con default 1

## Contexto

El analyze report necesita un schema JSON que siga las convenciones existentes: `version: 1`, `kind` discriminator (D1). Debe ser extensible para futuras categorias y consumible por un agent futuro.

## Alcance

**In**:
1. Definir structs Go en `internal/infer/report.go` (o similar):
   - `AnalyzeReport{Version, Kind, Path, Categories, Summary}`
   - `CategoryResult{ID, Name, InferenceCount, Inferences}`
   - `Inference{Type, Field, Value, Confidence, RequiresAgent, Evidence}`
2. Metodo `MarshalJSON` si necesario para formato especifico
3. Enum de tipos de inferencia como constantes Go

**Out**: Implementar el comando CLI (T002). Poblar el report con datos reales (T002).

## Estado inicial esperado

- No existen structs de report en internal/infer/
- Convenciones JSON: todos los outputs llevan `version: 1` (D1)

## Criterios de Aceptacion

- AnalyzeReport compila con `Version: 1, Kind: "analyze"`
- JSON output es valido y parseable
- Inference struct tiene campo `RequiresAgent bool`
- Test unitario: marshal → unmarshal roundtrip exitoso
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- Contratos JSON existentes: `internal/mcp/` tools, `cmd/rootline/table.go` output
- D1: additive-only, version: 1, kind discriminator
