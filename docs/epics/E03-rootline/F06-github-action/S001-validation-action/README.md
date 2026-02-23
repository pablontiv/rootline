---
estado: Pending
tipo: historia
cliente: Platform Owner
---
# S001: Validation Action

**Feature**: [F06 GitHub Action](../README.md)
**Capacidad**: rootline validate corre como step de CI y produce annotations visibles en PRs de GitHub

## Antes / Despues

**Antes**: rootline validate solo corre localmente. CI no verifica documentacion estructurada. Errores de schema pasan sin deteccion.

**Despues**: CI ejecuta rootline validate automaticamente. PRs con documentos invalidos muestran annotations de error. rootline's own CI valida docs/epics/ en cada push.

## Criterios de Aceptacion (semanticos)

- [ ] GitHub Action composite descarga rootline y ejecuta validate
- [ ] Errores de validacion aparecen como annotations en PR checks
- [ ] CI de rootline valida sus propios docs/epics/ en cada push

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-composite-action.md) | Crear composite action definition (action.yml) |
| [T002](T002-pr-annotations.md) | Integrar PR annotations via workflow commands |
| [T003](T003-dogfood-ci.md) | Agregar validacion de docs a CI de rootline |

## Fuente de verdad

- `.github/workflows/ci.yml`
- `action.yml`
