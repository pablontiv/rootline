---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar Cat 12 invariant extraction via regex + AST

**Story**: [S002 Body-Aware Categories 6/12/13](README.md)
**Contribuye a**: Cat 12 extrae invariantes del body

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Cat 12 detecta invariantes documentados en el body con patron `INV\d+:` o `- INV\d+:`. Los invariantes suelen estar en secciones `## Invariantes` o `## Preserva`. AST permite localizar la seccion correcta antes de aplicar regex, evitando false positives en code blocks o ejemplos.

## Alcance

**In**:
1. Detector que recibe Record con AST
2. Localiza secciones `Invariantes` o `Preserva` via ExtractSections
3. Aplica regex `INV\d+:` dentro de esas secciones
4. Extrae texto del invariante (linea completa despues de `INV\d+:`)
5. Produce inferencias de tipo `invariant_declaration` con ID y texto
6. Cross-reference: si invariante referencia un invariante de un nivel superior (heredado), marcar como `inherited_invariant`

**Out**: Detectar redundancia entre invariantes (eso es semantico, agent territory). Validar que invariantes se cumplen (eso es runtime).

## Estado inicial esperado

- F01 completado (ExtractSections disponible)
- Documentos existentes usan patron `INV\d+:` (ej: docs/epics/ Story READMEs)

## Criterios de Aceptacion

- Record con `## Invariantes\n- INV1: tests pass` → produce `invariant_declaration{id: "INV1", text: "tests pass"}`
- `INV1:` en code block → NO se extrae (AST filtra code blocks)
- Record sin seccion Invariantes → retorna []Inference vacio
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/extract/body.go` — ExtractSections
- Documentos con invariantes: `docs/epics/E12-v1-stem-removal/F01-hard-reject-v1/S001-engine-rejects-v1/README.md`
