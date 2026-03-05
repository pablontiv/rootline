---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Implementar Cat 13 sub-schema detection per type group

**Story**: [S002 Body-Aware Categories 6/12/13](README.md)
**Contribuye a**: Cat 13 detecta sub-schemas por valor de tipo

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Cat 13 detecta que records con diferente `tipo` tienen diferentes campos. Ejemplo: tasks con `tipo: test` tienen campo `ejecutable_en` pero tasks con `tipo: implementación` pueden no tenerlo. Esto se implementa agrupando records por valor de un campo discriminador y corriendo Analyze() por subgrupo para detectar FieldStats diferentes.

## Alcance

**In**:
1. Detector que recibe []Record de un directorio + campo discriminador (default: `tipo`)
2. Agrupa records por valor del discriminador
3. Corre FieldStats por subgrupo (reusar logica de Analyze)
4. Compara schemas entre subgrupos: campos exclusivos de un tipo → `type_specific_field`
5. Campos presentes en todos los tipos → `common_field`
6. Produce inferencias de tipo `sub_schema` con discriminador, valor, y campos exclusivos

**Out**: Generar .stem con `match:` por tipo (eso seria apply). Decidir si un campo "deberia" ser requerido por tipo (eso es semantico).

## Estado inicial esperado

- infer.Analyze() produce FieldStats con Values map
- Documentos existentes tienen campo `tipo` con valores variados

## Criterios de Aceptacion

- 5 records: 3 con `tipo: test` todos tienen campo `ejecutable_en`, 2 con `tipo: implementación` no lo tienen → produce `type_specific_field{discriminator: "tipo", value: "test", field: "ejecutable_en"}`
- Grupo con 1 solo record no produce inferencias (insufficient data)
- Sin campo discriminador en records → retorna []Inference vacio
- `go test ./internal/infer/ -race` pasa verde

## Fuente de verdad

- `internal/infer/infer.go` — Analyze(), FieldStats
- `internal/rules/match.go` — FilterSchemaByMatch (referencia para match-based filtering)
- Documentos reales: tasks en docs/epics/ tienen `tipo` con valores variados
