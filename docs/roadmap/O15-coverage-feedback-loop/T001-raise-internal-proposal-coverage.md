---
estado: Completed
tipo: task
---
# T001: Subir internal/proposal a ≥85% coverage

**Outcome**: [O15 Coverage feedback loop](README.md)
**Contribuye a**: precondición para activar el gate per-package (INV2)

## Preserva

- INV2 del outcome: el piso es 85% uniforme; ningún paquete recibe descuento
  - Verificar: `go test ./internal/proposal/ -cover` reporta ≥ 85.0%

## Contexto

`internal/proposal` está en 67.3% (medido `2026-05-21`). Las funciones de menor cobertura son detectores privados llamados solo cuando `Analyze` enfrenta inputs específicos:

| Función | Coverage actual | Naturaleza |
|---|---|---|
| `detectCorrectLink` (proposal.go:525) | 6.7% | recibe broken-link reports + records; emite CorrectLink |
| `detectMigrateValue` (proposal.go:293) | 34.2% | detecta enum→canonical mapping (ej. "Inprogres"→"In Progress") |
| `detectExtractBody` (proposal.go:394) | 27.6% | extrae H1/sección a frontmatter |
| `Analyze` (proposal.go:100) | 45.8% | orquestador top-level que llama detectores |
| `DetectMissingAggregates` (proposal.go:647) | 0% (parcial post-PR #36) | propone aggregate exprs faltantes |

Son funciones puras: input → propuestas. Tests de tabla con records sintéticos (`extract.Record` con `Frontmatter`, `Body`, `Path`) y stems mínimos (`rules.StemFile` con schema enum). El paquete ya tiene `proposal_test.go`, `parse_test.go`, `surface_test.go`, `evolution_test.go`, etc. — el patrón existe.

## Alcance

**In**:
1. Tests adicionales en `internal/proposal/` para `detectCorrectLink`, `detectMigrateValue`, `detectExtractBody`, `Analyze` y `DetectMissingAggregates` cubriendo casos diversos.
2. Cubrir tanto el happy path como ramas de rechazo (record sin field, stem nil, sin candidatos en records).
3. Estructurar como tests de tabla cuando sea natural.

**Out**:
- No modificar lógica de los detectores (sin refactor "while I'm here").
- No tocar otros paquetes — la subida es exclusivamente en `internal/proposal/`.
- No cambiar `proposal.Type` ni el contrato JSON.

## Estado inicial esperado

- `go test ./internal/proposal/ -cover` reporta ~67% en master `ca345a3` o posterior.
- Existen `internal/proposal/proposal.go` y test files actuales.

## Criterios de Aceptación

- `go test ./internal/proposal/ -cover` reporta `coverage: 85.0% of statements` o superior.
- `go test ./... -race` sigue verde.
- `golangci-lint run` sin issues nuevos.
- Diff incluye solo archivos `_test.go` y `testdata/` en `internal/proposal/`.

## Fuente de verdad

- `/home/shared/rootline/internal/proposal/proposal.go` — funciones target
- `/home/shared/rootline/internal/proposal/surface.go` — clasificación
- `/home/shared/rootline/internal/proposal/*_test.go` — patrones de test existentes
- `/home/shared/rootline/internal/extract/record.go` — struct Record
- `/home/shared/rootline/internal/rules/rules.go` — StemFile / SchemaField
