---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar body section structure analysis

**Story**: [S002 Body-Aware Inference](README.md)
**Contribuye a**: Body-section detector detecta patrones de heading structure

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Este detector analiza la estructura de secciones del body para detectar patrones: ¿todos los docs de un directorio tienen las mismas secciones? ¿Hay secciones requeridas? Usa ExtractSections de F01/S002 para obtener headings del AST.

## Alcance

**In**:
1. Detector que recibe []Record (con AST) de un directorio
2. Extrae secciones de cada Record via ExtractSections
3. Calcula consensus: secciones que aparecen en ≥80% de records → `required_section`
4. Secciones que aparecen en <20% → `optional_section`
5. Produce inferencias de tipo `section_pattern` con heading name y frequency

**Out**: Validar contenido de secciones (eso es semantico, agent territory). Sugerir template de body (futuro).

## Estado inicial esperado

- F01 completado (ExtractSections disponible)
- No hay detector de section patterns

## Criterios de Aceptacion

- Directorio donde 5/5 records tienen `## Contexto` → produce `required_section{heading: "Contexto", frequency: 1.0}`
- Directorio donde 1/5 records tiene `## Notas` → produce `optional_section{heading: "Notas", frequency: 0.2}`
- Records sin AST (body vacio) se ignoran sin error
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/extract/body.go` — ExtractSections
- Docs existentes en docs/epics/ como ejemplos reales de section patterns
