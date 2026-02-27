---
ejecutable_en: 1 sesion
estado: Completed
tipo: documentation
---
# T002: Reemplazar pre-commit-golang archivado con alternativa mantenida

**Story**: [S001 Cleanup and Standards](README.md)

## Contexto

`.pre-commit-config.yaml` usa `dnephin/pre-commit-golang` (v0.5.1) para el hook `go-fmt`. Este repositorio está archivado en GitHub (read-only, sin mantenimiento). Depender de un proyecto archivado es un riesgo: si pre-commit cambia su API o Go cambia algo, nadie actualizará el hook. La alternativa es usar un hook local definido inline en `.pre-commit-config.yaml` que ejecute `gofmt` directamente, eliminando la dependencia externa.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - commit (via pre-commit hook)
jobs:
  - nombre: replace-precommit-hook
    pasos:
      - Eliminar repo dnephin/pre-commit-golang de .pre-commit-config.yaml
      - Agregar hook local que ejecute gofmt -l -w
artefactos:
  - .pre-commit-config.yaml (modificado)
```

## Dependencias

- Ninguna

## Alcance

**In**:
1. En `.pre-commit-config.yaml`, eliminar el bloque de `dnephin/pre-commit-golang`
2. Agregar hook local para `gofmt`:
   ```yaml
   - repo: local
     hooks:
       - id: go-fmt
         name: gofmt
         entry: gofmt -l -w
         language: system
         types: [go]
   ```
3. Verificar que `pre-commit run --all-files` pasa

**Out**: Cambiar versión de golangci-lint en pre-commit, agregar hooks adicionales (go mod tidy, etc.)

## Estado inicial esperado

- `.pre-commit-config.yaml` tiene 2 repos: golangci-lint y dnephin/pre-commit-golang
- `dnephin/pre-commit-golang` está archivado en GitHub

## Criterios de Aceptacion

- `.pre-commit-config.yaml` no contiene referencia a `dnephin/pre-commit-golang`
- `.pre-commit-config.yaml` contiene hook local `go-fmt` que ejecuta `gofmt`
- `pre-commit run go-fmt --all-files` pasa sin errores
- `pre-commit run --all-files` pasa (ambos hooks: golangci-lint + go-fmt local)

## Fuente de verdad

- `.pre-commit-config.yaml` — a modificar
