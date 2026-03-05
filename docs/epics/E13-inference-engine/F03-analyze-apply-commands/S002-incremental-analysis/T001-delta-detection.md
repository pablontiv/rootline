---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar delta detection entre .stem e inferencias

**Story**: [S002 Incremental Analysis Mode](README.md)
**Contribuye a**: Delta detection compara inferencias contra .stem existente

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

drift.go ya detecta drift entre .stem y datos. El delta detection para analyze es similar pero opera sobre inferencias: compara cada inferencia contra el .stem actual para determinar si ya esta cubierta (campo ya requerido, enum ya incluye valor, etc.).

## Alcance

**In**:
1. Funcion `FilterCoveredInferences(inferences []Inference, stem StemSchema) []Inference`
2. Inferencia `add_required_field{field: "estado"}` cubierta si stem ya tiene `required: [estado]`
3. Inferencia `extend_enum{field: "tipo", value: "test"}` cubierta si stem ya tiene `test` en enum
4. Inferencia `constant_field{field: "X", value: "Y"}` cubierta si stem tiene default o required match
5. Retorna solo inferencias NO cubiertas (deltas)

**Out**: Integrar con flag --incremental (T002). Apply de deltas (S003).

## Estado inicial esperado

- AnalyzeReport y detectores producen inferencias
- StemSchema cargable via rules.LoadStem

## Criterios de Aceptacion

- Inferencia cubierta por .stem → filtrada (no aparece en output)
- Inferencia no cubierta → presente en output
- Test: .stem con `required: [estado]` + inferencia `add_required_field{estado}` → filtrada
- Test: .stem sin required + inferencia `add_required_field{estado}` → presente
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/rules/drift.go` — DetectDrift como referencia
- `internal/rules/rules.go` — StemSchema, field definitions
