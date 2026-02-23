---
estado: Completed
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T002: Agregar verificación de go mod tidy en CI

**Story**: [S001 Pipeline Hardening](README.md)

## Contexto

Si un developer agrega un import y no corre `go mod tidy`, el `go.mod`/`go.sum` puede quedar desincronizado. Esto no causa build failures inmediatos pero puede causar problemas en otros environments. La verificación en CI ejecuta `go mod tidy` y falla si produce diferencias, asegurando que el módulo siempre está limpio.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - push
  - pr
jobs:
  - nombre: tidy-check (nuevo)
    pasos:
      - Checkout code
      - Setup Go
      - Run go mod tidy
      - git diff --exit-code go.mod go.sum
artefactos:
  - .github/workflows/ci.yml (modificado)
```

## Dependencias

- Ninguna

## Alcance

**In**:
1. Agregar job `tidy` a `.github/workflows/ci.yml`
2. El job ejecuta: `go mod tidy` seguido de `git diff --exit-code go.mod go.sum`
3. Si hay diff, el job falla con mensaje indicando que `go mod tidy` no fue ejecutado

**Out**: Agregar `go mod tidy` a pre-commit hooks, auto-fix en CI

## Estado inicial esperado

- `ci.yml` tiene 3 jobs: build, test, lint
- No hay verificación de go mod tidy

## Criterios de Aceptacion

- `.github/workflows/ci.yml` contiene job que ejecuta `go mod tidy` y verifica que no hay diff
- El job usa `git diff --exit-code go.mod go.sum` para detectar cambios
- Workflow YAML es válido

## Fuente de verdad

- `.github/workflows/ci.yml` — a modificar
- `go.mod`, `go.sum` — archivos verificados
