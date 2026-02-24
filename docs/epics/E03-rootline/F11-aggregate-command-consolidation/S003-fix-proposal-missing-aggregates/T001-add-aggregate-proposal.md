---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Add Aggregate Proposal to Fix Engine

**Story**: [S003 Fix Proposal for Missing Aggregates](README.md)
**Contribuye a**: `rootline fix` detecta y aplica AddAggregate

[[blocks:T001-implement-aggregate-generator]]

## Preserva

- INV1: Tests existentes siguen pasando
  - Verificar: `go test ./cmd/rootline/ -run TestFix -v`
- INV2: fix no corrompe .stem existentes
  - Verificar: `rootline validate` después de fix en test

## Contexto

El proposal engine (`internal/proposal/proposal.go`) tiene 7 tipos de proposals con `Analyze()` ejecutando fases de detección secuencial. `cmd/rootline/fix.go` tiene `applyProposals()` que ya maneja `ExtendEnum` via YAML AST (`addEnumValueToNode`). El nuevo `AddAggregate` sigue el mismo patrón: detectar en proposal.go, aplicar en fix.go.

La diferencia clave: `ExtendEnum` modifica un nodo existente del YAML, mientras que `AddAggregate` necesita **agregar una sección nueva** (`aggregate:`) al .stem. El patrón de YAML AST manipulation ya existe en `addEnumValueToNode` — `addAggregateToStem` necesita crear un mapping node nuevo a nivel root del documento.

## Especificacion Tecnica

**Modificar**: `internal/proposal/proposal.go`
- Nuevo `ProposalType`: `AddAggregate`
- Nuevo campo en `Proposal`: `AggregateExpr string` (expresión generada)
- Nueva función `detectAddAggregate(dir string, stems []StemInfo, records []Record) []Proposal`:
  1. Usar `infer.AnalyzeHierarchy(records)` para detectar si es jerarquía
  2. Si no es jerarquía → retornar nil
  3. Para cada campo enum en root schema del stem
  4. Si no existe aggregate para ese campo → generar con `migrate.GenerateAggregateExpr()`
  5. Crear `Proposal{Type: AddAggregate, StemPath: ..., Field: fieldName, AggregateExpr: expr}`
- Registrar `detectAddAggregate` como nueva fase en `Analyze()` (después de las fases existentes)

**Modificar**: `cmd/rootline/fix.go`
- Nuevo case `proposal.AddAggregate` en `applyProposals()`:
  1. Leer .stem file como YAML AST (`yaml.Node`)
  2. Buscar nodo `aggregate:` en el mapping root
  3. Si no existe → crear mapping node `aggregate:` con el campo
  4. Si existe → agregar key-value al mapping existente
  5. Escribir YAML actualizado
- Función `addAggregateToStem(stemPath, fieldName, expr string) error`

**Tests**: `cmd/rootline/fix_test.go` — 2 tests nuevos
1. `TestFix_AddAggregate_DryRun`: directorio con enum jerárquico sin aggregate → dry-run reporta propuesta con expresión
2. `TestFix_AddAggregate_Apply`: directorio con enum jerárquico sin aggregate → fix aplica, .stem resultante tiene `aggregate:` section, `rootline validate` pasa

## Dependencias

> Requiere T001-implement-aggregate-generator (S001) completado.

- `internal/migrate` — `GenerateAggregateExpr()` function
- `internal/infer` — `AnalyzeHierarchy()` function
- `internal/proposal` — existing Proposal types and Analyze() pattern

## Alcance

**In**:
1. Agregar `AddAggregate` type y `detectAddAggregate()` en `proposal.go`
2. Agregar case `AddAggregate` y `addAggregateToStem()` en `fix.go`
3. Agregar 2 tests en `fix_test.go`

**Out**: No tocar `validate.go`, `doctor.go`, `init.go`, `migrate.go`.

## Estado inicial esperado

- `internal/migrate/aggregate.go` existe con `GenerateAggregateExpr()` funcional
- `internal/proposal/proposal.go` tiene 7 ProposalTypes y Analyze() con fases
- `cmd/rootline/fix.go` tiene `applyProposals()` con case `ExtendEnum` como patrón

## Criterios de Aceptacion

- `go test ./cmd/rootline/ -run TestFix -v` pasa (tests existentes + 2 nuevos)
- `rootline fix --all --dry-run` sobre directorio con enum jerárquico sin aggregate → reporta "would add aggregate for 'estado'"
- `rootline fix --all` sobre mismo directorio → produce .stem con `aggregate:` section válido
- `rootline validate` sobre .stem modificado → valid: true

## Fuente de verdad

- `internal/proposal/proposal.go` — ProposalType enum, Analyze() (~64), Proposal struct
- `cmd/rootline/fix.go` — applyProposals (~280), addEnumValueToNode (patrón YAML AST)
- `internal/infer/hierarchy.go` — AnalyzeHierarchy
- `internal/migrate/aggregate.go` — GenerateAggregateExpr (creado en S001/T001)
