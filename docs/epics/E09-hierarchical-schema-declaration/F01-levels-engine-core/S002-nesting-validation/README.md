---
estado: Completed
---
# S002: Nesting Validation

**Feature**: [F01 Levels Engine Core](../README.md)
**Capacidad**: `rootline validate` detecta records ubicados en niveles incorrectos de la jerarquia
**Cubre**: P2 del Epic — nesting violations detectadas por validate

## Antes / Despues

**Antes**: Un task file puede existir directamente bajo un directorio de epic sin que rootline lo detecte como error. No hay enforcement de la jerarquia E→F→S→T.

**Despues**: Si un `.stem` define `levels:` con `children:` constraints, `rootline validate` reporta error cuando un record esta en un nivel que no es hijo permitido de su padre. Ejemplo: `T001.md` directo bajo `E01/` genera error porque `epic.children = [feature]`.

## Criterios de Aceptacion (semanticos)

- [ ] `CheckNesting` valida cadena E→F→S→T como correcta
- [ ] `CheckNesting` detecta task directo bajo epic como error
- [ ] `CheckNesting` detecta subdir bajo leaf level (children: []) como error
- [ ] `CheckNesting` sin levels no genera errores (skip transparente)
- [ ] Nesting errors se integran en validate y fix pipelines

## Invariantes

- INV1: Todos los tests existentes pasan sin modificacion
  - Verificar: `go test ./... -race`
- INV3: `.stem` sin `levels:` zero regression
  - Verificar: tests sin levels no generan nesting errors

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-implement-checknesting-function.md) | Implement CheckNesting function |
| [T002](T002-integrate-nesting-check-in-pipelines.md) | Integrate nesting check in validate and fix pipelines |

## Fuente de verdad

- `internal/rules/hierarchy.go` — (nuevo) ExpandLevels, CheckNesting
- `cmd/rootline/validate.go` — validate pipeline
- `cmd/rootline/fix.go` — fix pipeline
