---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar Cat 11 engine portion — traceability link extraction

**Story**: [S003 Semantic Category Stubs 9/11](README.md)
**Contribuye a**: Cat 11 extrae claims de traceability del body

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Cat 11 (traceability) tiene 20% Go / 80% LLM. La porcion engine extrae claims explicitos de traceability: "Contribuye a", "Cubre", "Satisface" en el body. El semantic matching (¿realmente contribuye a ese objetivo?) queda para el agent.

## Alcance

**In**:
1. Detector que recibe Record con body
2. Regex para patrones de traceability: `Contribuye a:`, `Cubre:`, `Satisface:`, `**Satisface**:`
3. Extrae target del claim (texto despues del patron)
4. Produce inferencias de tipo `traceability_claim` con source, verb, y target
5. Si target es un path o wiki-link resolvible → `verified_traceability`
6. Si target es texto libre → `unverified_traceability{requires_agent: true}`

**Out**: Semantic matching de traceability (agent Epic). Validar que claims son correctos.

## Estado inicial esperado

- F01 completado (AST disponible)
- Documentos existentes usan patron `**Contribuye a**:` y `**Cubre**:`

## Criterios de Aceptacion

- `**Contribuye a**: tests pass` → produce `traceability_claim{verb: "contribuye_a", target: "tests pass"}`
- `**Satisface**: P1, P3` → produce 2 claims separados (P1 y P3)
- Target que es wiki-link `[[E13/F01]]` → `verified_traceability`
- Target que es texto libre → `unverified_traceability{requires_agent: true}`
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- Documentos con traceability: Feature READMEs (`**Satisface**: P1`), Task files (`**Contribuye a**:`)
- Patrones existentes en docs/epics/
