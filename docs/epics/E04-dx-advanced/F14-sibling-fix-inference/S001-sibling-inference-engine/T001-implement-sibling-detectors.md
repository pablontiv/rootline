---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implement detectInferFromSiblings and detectOutlierValue

**Story**: [S001 Sibling Inference Engine](README.md)
**Contribuye a**: fix propone infer_from_siblings y correct_outlier con valores correctos

## Preserva

- INV1: Tests existentes pasan sin cambios
  - Verificar: `go test ./... -race`
- INV2: Coverage >= 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

El proposal engine de rootline (`internal/proposal/`) genera propuestas categorizadas para corregir errores de validacion. Actualmente, cuando un campo enum required falta, `detectAddField` propone el primer valor del enum como default — frecuentemente incorrecto. No existe detector para valores validos pero semanticamente incorrectos (outliers).

La solucion es agrupar records por directorio padre (siblings) y usar la mayoria estadistica para inferir el valor correcto.

## Alcance

**In**:
1. Crear `internal/proposal/sibling_infer.go` con tres funciones:
   - `majorityValue(values []string) (string, int, int)` — modo estadistico
   - `detectInferFromSiblings(records, effective, errs)` — para campos faltantes
   - `detectOutlierValue(records, effective, errs)` — para valores outlier
2. Crear `internal/proposal/sibling_infer_test.go` con ~10 tests

**Out**: Integracion en Analyze pipeline (T002), cambios a fix engine (T002), cambios a CLI (T002)

## Especificacion Tecnica

### Constantes

```go
siblingInferThreshold = 0.6   // 60% agreement for missing fields
outlierThreshold      = 0.75  // 75% agreement to flag outliers
minSiblingsForInfer   = 2     // minimum siblings with majority value
minSiblingsForOutlier = 3     // higher bar for corrections
```

### detectInferFromSiblings

1. Agrupar records por `filepath.Dir(path)`, excluir README.md
2. Para cada campo enum en `effective.Schema`:
   a. Recolectar valores no-vacios del `Frontmatter` de siblings
   b. Calcular `majorityValue()`
   c. Si ratio >= 0.6 Y count >= 2:
      - Para records en `errs` con `rule == "required"` en ese campo
      - Generar `Proposal{Type: InferFromSiblings, Field, Value: majority, Paths: [path]}`
      - Description: `"infer %q from %d siblings (%d/%d agree)"`

### detectOutlierValue

1. Misma agrupacion
2. Para cada campo enum, encontrar records que:
   - Tienen el campo en frontmatter (valor no-vacio)
   - NO tienen errores de validacion para ese campo
   - Su valor difiere de la mayoria
3. Si majority ratio >= 0.75 Y count >= 3:
   - Generar `Proposal{Type: CorrectOutlier, Field, From: current, To: majority, Paths: [path]}`
   - Description: `"value %q is an outlier among %d siblings (majority: %q, %d/%d)"`

### Tests requeridos

| Test | Escenario |
|------|-----------|
| TestMajorityValue_Clear | Un valor dominante |
| TestMajorityValue_Tie | Empate retorna cualquiera o vacio |
| TestMajorityValue_Empty | Slice vacio retorna ("", 0, 0) |
| TestInferFromSiblings_MajorityPresent | 3 siblings con mismo valor, 1 missing → propone |
| TestInferFromSiblings_BelowThreshold | Ratio < 0.6 → no propone |
| TestInferFromSiblings_SkipsREADME | README.md excluido del grouping |
| TestInferFromSiblings_OnlyEnumFields | Campos non-enum ignorados |
| TestInferFromSiblings_CrossDirectoryIsolation | Siblings de distintos dirs no se mezclan |
| TestOutlierValue_StrongConsensus | 5 con A, 1 con B → propone corregir B a A |
| TestOutlierValue_WeakConsensus | 3 con A, 2 con B (60% < 75%) → no propone |
| TestOutlierValue_MinimumSiblings | Solo 2 siblings agree → no propone |
| TestOutlierValue_SkipsRecordsWithErrors | Records con errores de validacion no se flagean como outliers |

## Criterios de Aceptacion

- `go test ./internal/proposal/ -run TestMajority -v` — all pass
- `go test ./internal/proposal/ -run TestInferFromSiblings -v` — all pass
- `go test ./internal/proposal/ -run TestOutlierValue -v` — all pass
- `go vet ./internal/proposal/` — no issues

## Fuente de verdad

- `internal/proposal/proposal.go` — Proposal struct, Type constants (para referencia)
- `internal/proposal/infer.go` — patron existente de inference (InferEstado, mapValue)
- `internal/extract/extract.go` — Record struct con Frontmatter map
- `internal/rules/rules.go` — StemFile, SchemaField structs
