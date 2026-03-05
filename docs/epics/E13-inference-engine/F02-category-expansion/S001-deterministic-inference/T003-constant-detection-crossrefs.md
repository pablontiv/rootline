---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Implementar Cat 8 constant detection y Cat 10 cross-epic refs

**Story**: [S001 Deterministic Categories 5/7/8/10](README.md)
**Contribuye a**: Cat 8 detecta campos constantes; Cat 10 extrae path references

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Cat 8: `infer.Analyze()` ya calcula FieldStats con presence y value frequency. Un campo donde 100% de records tienen el mismo valor es una constante — deberia estar en .stem como default o required value. Cat 10: Body content contiene path references como `E02/F04/S001` que apuntan a otros artefactos del roadmap — extraerlos y validar que existen en el filesystem.

## Alcance

**In**:
1. Cat 8: Detector que analiza FieldStats — si un campo tiene 100% mismo valor en ≥3 records → `constant_field`
2. Cat 10: Regex `[EFS]\d{2,3}(/[EFS]\d{2,3})*` sobre body para extraer path references
3. Cat 10: Validar que paths extraidos existen relativos al directorio raiz
4. Cat 10: Produce inferencias de tipo `cross_reference` (validas) y `broken_cross_reference` (no existen)

**Out**: Aplicar constantes al .stem (eso es apply). Resolver ambiguedades de paths (trivial, no necesita LLM).

## Estado inicial esperado

- infer.Analyze() produce FieldStats con Values map
- Record.Body disponible como string

## Criterios de Aceptacion

- Cat 8: Campo `estado: Specified` en 5/5 records → produce `constant_field{field: "estado", value: "Specified"}`
- Cat 8: Campo con 2 valores distintos → no produce inferencia
- Cat 10: Body con `[E02/F04](../E02/F04/)` → extrae `E02/F04` como cross-reference
- Cat 10: Path que no existe → produce `broken_cross_reference`
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/infer/infer.go` — Analyze(), FieldStats con Values map
- Docs existentes con cross-references: `docs/epics/` (ejemplos reales)
