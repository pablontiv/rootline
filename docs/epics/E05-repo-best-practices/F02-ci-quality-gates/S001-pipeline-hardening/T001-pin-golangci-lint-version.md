---
estado: Completed
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T001: Pinear golangci-lint a versión explícita en CI

**Story**: [S001 Pipeline Hardening](README.md)

## Contexto

El workflow CI usa `version: latest` para golangci-lint, lo que significa que cada run puede usar una versión diferente. Esto causa builds no reproducibles — un PR puede fallar por un linter nuevo que no existía cuando el código se escribió. El pre-commit ya pinea `v2.1.6`. CI debe usar la misma versión para consistencia.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - push
  - pr
jobs:
  - nombre: lint (existente, modificar)
    pasos:
      - Cambiar version de latest a v2.1.6 en golangci-lint-action
artefactos:
  - .github/workflows/ci.yml (modificado)
```

## Dependencias

- Ninguna

## Alcance

**In**:
1. En `.github/workflows/ci.yml`, cambiar `version: latest` a `version: v2.1.6` en el step de `golangci/golangci-lint-action`
2. Verificar que la versión coincide con `.pre-commit-config.yaml` rev

**Out**: Actualizar la versión de golangci-lint (solo pinear la existente), crear proceso de actualización sincronizada

## Estado inicial esperado

- `ci.yml` lint job usa `version: latest` en golangci-lint-action
- `.pre-commit-config.yaml` usa `rev: v2.1.6`

## Criterios de Aceptacion

- `.github/workflows/ci.yml` contiene `version: v2.1.6` (no `latest`)
- La versión coincide con `.pre-commit-config.yaml` rev
- Workflow YAML es válido

## Fuente de verdad

- `.github/workflows/ci.yml` — a modificar
- `.pre-commit-config.yaml` — referencia de versión
