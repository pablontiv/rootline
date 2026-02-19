# S002: CI Pipeline

**Feature**: [F01 Project Foundation](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Cada push al repositorio ejecuta build, test y lint automaticamente

## Antes / Despues

**Antes**: No hay verificacion automatica de calidad. Errores de compilacion o estilo se detectan manualmente.

**Despues**: GitHub Actions ejecuta `go build`, `go test`, y `golangci-lint` en cada push y PR. PRs no se pueden mergear con checks fallidos.

## Criterios de Aceptacion (semanticos)

- [ ] Push a main trigger CI pipeline exitosamente
- [ ] PR con test fallido muestra check rojo
- [ ] Linting detecta issues de estilo automaticamente

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-github-actions-workflow.md) | Crear GitHub Actions workflow para build, test, lint |
| [T002](T002-linting-config.md) | Configurar golangci-lint y pre-commit hooks |

## Fuente de verdad

- `.github/workflows/` — CI workflows
- `.golangci.yml` — linter config
