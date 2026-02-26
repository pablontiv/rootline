---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Integrate sibling inference into Analyze pipeline and fix engine

**Story**: [S001 Sibling Inference Engine](README.md)
**Contribuye a**: fix engine aplica propuestas infer_from_siblings y correct_outlier

[[blocks:T001-implement-sibling-detectors]]

## Preserva

- INV1: Tests existentes pasan sin cambios
  - Verificar: `go test ./... -race`
- INV2: Coverage >= 85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
- INV3: `infer_from_siblings` tiene prioridad sobre `add_field` pero no sobre `extract_body` ni `infer_from_children`
  - Verificar: test de integracion en Analyze

## Contexto

T001 implemento `detectInferFromSiblings` y `detectOutlierValue` como funciones independientes con tests unitarios. Este task los integra en el pipeline de `Analyze()`, el fix engine, y el CLI output para que `rootline fix --all` los use automaticamente.

## Alcance

**In**:
1. Agregar tipos `InferFromSiblings` y `CorrectOutlier` a proposal.go
2. Integrar detectores en `Analyze()` pipeline con prioridad correcta
3. Agregar cases en `ApplyProposals()` en fix.go
4. Agregar cases en `proposalsToFixResults()` en cmd/rootline/fix.go
5. Agregar e2e test

**Out**: Cambios a los detectores (ya implementados en T001), cambios a .stem schema

## Especificacion Tecnica

### proposal.go cambios

```go
// Nuevos tipos
InferFromSiblings Type = "infer_from_siblings"
CorrectOutlier    Type = "correct_outlier"

// Summary fields
InferFromSiblings int `json:"infer_from_siblings"`
CorrectOutlier    int `json:"correct_outlier"`
```

En `Analyze()`, insertar despues de Phase 3 (linea ~120), antes de `detectAddField`:

```go
// Phase 3b: sibling inference has priority over add_field
siblingProposals := detectInferFromSiblings(records, effective, errs)
proposals = append(proposals, siblingProposals...)
for _, p := range siblingProposals {
    for _, path := range p.Paths {
        coveredKeys[path+"\x00"+p.Field] = true
    }
}
```

Despues de Phase 4, independientemente:

```go
// Phase 5: outlier detection (independent — valid-but-wrong values)
proposals = append(proposals, detectOutlierValue(records, effective, errs)...)
```

### fix.go cambios (internal/fix/)

En `ApplyProposals` switch, agregar:

```go
case proposal.InferFromSiblings:
    // Same path as AddField — set field value in frontmatter
    // Add to existing ExtractBody/InferFromChildren/AddField case
case proposal.CorrectOutlier:
    if err := applyCorrectValue(p, root, recordMap); err != nil {
        return fmt.Errorf("correct_outlier %s: %w", p.Paths[0], err)
    }
```

### cmd/rootline/fix.go cambios

En `proposalsToFixResults`, agregar `InferFromSiblings` al case de AddField (linea ~294):

```go
case proposal.AddField, proposal.ExtractBody, proposal.InferFromChildren, proposal.InferFromSiblings:
```

Agregar `CorrectOutlier` al case de CorrectValue (linea ~297):

```go
case proposal.CorrectValue, proposal.MigrateValue, proposal.CorrectLink, proposal.CorrectOutlier:
```

### E2E test

Agregar `TestFixAllSiblingInference` en `internal/e2e/`:
1. Crear temp dir con .stem que define campo enum `tipo` con values [a, b, c]
2. Crear 4 files: 3 con `tipo: a`, 1 sin `tipo`
3. Ejecutar fix --all --dry-run programaticamente
4. Verificar que proposal es `infer_from_siblings` con value `a`

## Criterios de Aceptacion

- `go test ./... -race` — all pass
- Coverage >= 85%
- `rootline fix --all --dry-run /opt/homeserver/automation/docs/epics/` → 0 proposals (data ya corregida, confirma ausencia de falsos positivos)

## Fuente de verdad

- `internal/proposal/proposal.go:71-180` — Analyze pipeline
- `internal/fix/fix.go:25-110` — ApplyProposals
- `cmd/rootline/fix.go:270-338` — proposalsToFixResults
- `internal/proposal/sibling_infer.go` — detectores creados en T001
