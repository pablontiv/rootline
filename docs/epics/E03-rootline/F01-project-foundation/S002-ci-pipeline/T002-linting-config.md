---
estado: Pending
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T002: Configurar golangci-lint y pre-commit hooks

**Story**: [S002 CI Pipeline](README.md)

## Contexto

golangci-lint es el metalinter estandar para Go. Configurarlo desde el inicio asegura estilo consistente y detecta problemas comunes. Pre-commit hooks complementan ejecutando checks antes de cada commit.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - push
  - pr
jobs:
  - nombre: lint-config
    pasos:
      - Crear .golangci.yml con linters seleccionados
      - Crear .pre-commit-config.yaml
      - Verificar que golangci-lint run pasa
artefactos: []
```

## Dependencias

- F01/S001 completado (Go module con codigo para lint)

## Alcance

**In**:
1. `.golangci.yml` con linters: govet, errcheck, staticcheck, unused, gosimple, ineffassign, gocritic
2. `.pre-commit-config.yaml` con hooks: golangci-lint, gofmt, go-vet
3. Timeouts y exclusiones razonables

**Out**: Custom linters, CI integration (es T001)

## Estado inicial esperado

- Go module existe con al menos un archivo .go
- golangci-lint instalable (`go install`)

## Criterios de Aceptacion

- `golangci-lint run` ejecuta sin errores en el proyecto actual
- `.golangci.yml` declara al menos 5 linters habilitados
- `.pre-commit-config.yaml` existe con hooks Go

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` seccion 7 (Stack: Testing)
