---
estado: Pending
tipo: feature
---
# F06: GitHub Action

**Epic**: [E03](../README.md)
**Objetivo**: Equipos pueden validar documentacion estructurada en CI con annotations en PRs
**Beneficio**: Enforcement automatico de schemas .stem en pipelines, equivalente a ESLint para documentacion
**Milestone**: PR con documento invalido muestra annotation de error en GitHub checks UI

## Scope

**In**: Composite GitHub Action, PR annotations via workflow commands, dogfood en CI de rootline
**Out**: Auto-fix en PRs, Docker action, marketplace publishing

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Validation Action](S001-validation-action/) | `rootline validate` corre en CI y anota errores en PRs |

## Dependencias

- rootline binary funcional (ya existe)

## Fuente de verdad

- `.github/workflows/ci.yml` (CI existente)
- `action.yml` (nueva action)
