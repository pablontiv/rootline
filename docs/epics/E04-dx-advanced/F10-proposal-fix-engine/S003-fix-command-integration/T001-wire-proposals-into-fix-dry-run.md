---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Wire Proposals into Fix Dry-Run

**Story**: [S003 Fix Command Integration](README.md)

[[blocks:T003-implement-body-and-child-inference-detectors]]

## Contexto

Con el package `internal/proposal/` completo (S002), el comando `rootline fix` necesita integrarlo para reemplazar el output plano del dry-run con el Report de propuestas categorizadas. `runFixAll()` actualmente valida archivos uno por uno y aplica fixes inline — necesita refactorearse para: (1) recopilar todos los records y errores primero, (2) llamar `proposal.Analyze()`, (3) outputear el Report.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: runFixAll (refactored)
    metodos:
      - nombre: runFixAll
        input: "cmd *cobra.Command"
        output: "error"
dependencias_externas: []
tests:
  - dry-run en epics de rootline con 0 errores → 0 proposals
  - dry-run JSON tiene kind "rootline/fix-proposals" y version 1
  - dry-run table muestra columnas Type, Description, Files, Resolves
```

## Alcance

**In**:
1. Refactorear `runFixAll()` en `cmd/rootline/fix.go`: primer pass recopila records + errores, segundo pass genera propuestas
2. En modo `--dry-run`: outputear `proposal.Report` como JSON (default) o table
3. Agregar `renderProposalTable()` para `--output table`
4. Sin `--dry-run`: mantener comportamiento actual temporalmente (T002 lo reemplaza)
5. Backward compatibility: JSON output cambia de `rootline/fix-batch` a `rootline/fix-proposals` solo en dry-run

**Out**: No implementar aplicacion de propuestas (eso es T002). No cambiar `runFix()` (single-file mode) todavia.

## Estado inicial esperado

- `cmd/rootline/fix.go` existe con `runFixAll()`, `applyFixes()`, `newBatchFixResult()`
- `internal/proposal/` existe con `Analyze()` que retorna `*Report`
- `go build ./cmd/rootline/` compila

## Criterios de Aceptacion

- `go build ./cmd/rootline/` compila sin errores
- `rootline fix --all --dry-run` en rootline's own epics → JSON con `"proposals": []`
- `rootline fix --all --dry-run -o table` muestra header "Type | Description | Files | Resolves"
- `rootline fix --all --dry-run` sin `--dry-run` mantiene comportamiento actual (no rompe nada)
- `go vet ./cmd/rootline/` sin errores

## Fuente de verdad

- `cmd/rootline/fix.go` — archivo a modificar
- `internal/proposal/proposal.go` — Report type a consumir
