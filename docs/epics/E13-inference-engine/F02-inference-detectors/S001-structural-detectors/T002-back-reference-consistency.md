---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar back-reference consistency check

**Story**: [S001 Structural Detectors](README.md)
**Contribuye a**: Back-reference consistency detecta links unidireccionales

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

`graph.Build()` en `internal/graph/graph.go` ya construye un grafo de dependencias entre documentos usando wiki-links. Este detector analiza el grafo para detectar links unidireccionales donde se espera reciprocidad — si A referencia B, ¿B referencia A?

## Alcance

**In**:
1. Detector que recibe el grafo de graph.Build()
2. Para cada edge A→B, verifica si existe edge B→A
3. Produce inferencias de tipo `missing_back_reference` cuando falta reciprocidad
4. Configurable: solo aplica a links de tipo `blocks`, `relates`, u otros que implican reciprocidad

**Out**: Crear back-references automaticamente (eso seria apply). Detectar broken links (ya existe en graph).

## Estado inicial esperado

- graph.Build() funciona y produce edges con source/target
- No hay detector de reciprocidad

## Criterios de Aceptacion

- Detector recibe Graph y retorna []Inference
- Link A→B sin B→A produce `missing_back_reference` con source=B, target=A
- Links que son inherently unidireccionales (ej: `blocks`) no producen false positives
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/graph/graph.go` — Build(), Edge struct
- `internal/extract/links.go` — Link.Type
