---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar link-type validation usando LinkSchema

**Story**: [S001 Structural Detectors](README.md)
**Contribuye a**: Link-type validation produce inferencias

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

LinkSchema en `internal/rules/rules.go` ya tiene campo `Allowed []string` pero nunca se valida. Este detector infiere que tipos de links son validos basado en los links observados en el directorio, y valida contra Allowed si esta definido.

## Alcance

**In**:
1. Detector que recibe links de todos los records de un directorio
2. Infiere `Allowed` link types si no estan definidos en .stem (consensus ≥80%)
3. Si `Allowed` esta definido, valida links contra la lista
4. Produce inferencias de tipo `link_type_suggestion` (nuevos) y `link_type_violation` (invalidos)

**Out**: Validacion de targets de links (eso es graph/broken links). Modificacion de LinkSchema struct.

## Estado inicial esperado

- LinkSchema struct existe con Allowed field (nunca validado)
- ParseLinks extrae links del body

## Criterios de Aceptacion

- Detector recibe []Record y retorna []Inference
- Con Allowed definido: link no en lista → produce `link_type_violation`
- Sin Allowed definido: infiere tipos observados con consensus ≥80%
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/rules/rules.go` — LinkSchema, Allowed field
- `internal/extract/links.go` — Link struct con Type field
