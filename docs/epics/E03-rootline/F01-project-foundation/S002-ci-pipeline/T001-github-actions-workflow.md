---
estado: Completado
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T001: Crear GitHub Actions workflow para build, test, lint

**Story**: [S002 CI Pipeline](README.md)

## Contexto

Rootline necesita CI desde el dia 1 para mantener quality gates. GitHub Actions es el estandar para proyectos Go open source. El workflow debe cubrir build, test y lint en cada push/PR.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - push
  - pr
jobs:
  - nombre: build
    pasos:
      - Setup Go 1.22+
      - go build ./...
  - nombre: test
    pasos:
      - go test ./... -race -coverprofile=coverage.out
      - Upload coverage report
  - nombre: lint
    pasos:
      - golangci-lint run
artefactos:
  - coverage.out
```

## Dependencias

- F01/S001 completado (Go module buildable)

## Alcance

**In**:
1. `.github/workflows/ci.yml` con jobs: build, test, lint
2. Matrix para Go versions (1.22, latest)
3. Cache de Go modules
4. Coverage report upload

**Out**: Release workflow (es F05/S002), deployment

## Estado inicial esperado

- Repositorio en GitHub con Go module funcional
- `go build ./...` y `go test ./...` exitosos localmente

## Criterios de Aceptacion

- Push a main ejecuta los 3 jobs (build, test, lint)
- Job `test` genera coverage report
- Job `lint` usa golangci-lint
- Workflow usa cache de Go modules

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` seccion 7 (Stack: CI/CD)
