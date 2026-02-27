---
estado: Completed
tipo: historia
---
# S001: Validate and Fix Path Support

**Feature**: [F12 CLI Path Argument Consistency](../README.md)
**Capacidad**: validate --all y fix --all aceptan directorio como argumento posicional
**Cubre**: P1 (validate path), P2 (fix path)

## Antes / Despues

**Antes**: `rootline validate --all docs/epics/` ignora `docs/epics/` y escanea desde CWD. Lo mismo con `fix --all`. Los skills documentan estos comandos como si funcionaran, pero el argumento de directorio no hace nada.

**Despues**: Ambos comandos usan `args[0]` como root cuando se proporciona, igual que los otros 4 comandos transversales (`tree`, `query`, `stats`, `graph`). Sin argumento, siguen usando CWD.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline validate --all docs/epics/` escanea solo docs/epics, no todo el repo
- [ ] `rootline fix --all docs/epics/ --dry-run` escanea solo docs/epics
- [ ] Sin argumentos, ambos siguen usando CWD (backward compatible)
- [ ] Tests existentes pasan

## Invariantes

- INV1: Sin argumentos, `--all` sigue usando CWD
  - Verificar: `rootline validate --all --output json | python3 -c "import json,sys; print(json.load(sys.stdin)['summary']['total'] > 0)"`
- INV2: Tests pasan
  - Verificar: `go test ./... -race`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-path-to-validate-all.md) | Add path argument to validate --all |
| [T002](T002-add-path-to-fix-all.md) | Add path argument to fix --all |

## Fuente de verdad

- `cmd/rootline/validate.go` — runValidateAll function
- `cmd/rootline/fix.go` — runFixAll function
- `cmd/rootline/tree.go:49-60` — patrón de referencia (scanRoot)
