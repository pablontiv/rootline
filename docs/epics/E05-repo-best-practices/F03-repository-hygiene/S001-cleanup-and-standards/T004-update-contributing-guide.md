---
estado: Completado
tipo: documentation
ejecutable_en: 1 sesion
---
# T004: Actualizar CONTRIBUTING.md con setup completo

**Story**: [S001 Cleanup and Standards](README.md)

## Contexto

`CONTRIBUTING.md` actual no menciona: instalación de pre-commit hooks, ejecución de `golangci-lint run ./...` (solo menciona `go vet ./...`), convención de conventional commits, ni la existencia de `.editorconfig`. Un nuevo contributor que siga la guía actual no tendría hooks de pre-commit activos y podría enviar commits con formato incorrecto.

## Dependencias

- T002 (pre-commit actualizado — para documentar los hooks correctos)
- T003 (editorconfig creado — para mencionarlo en la guía)

## Alcance

**In**:
1. Agregar sección "Pre-commit Hooks" en Development Setup:
   - `pip install pre-commit` (o `brew install pre-commit`)
   - `pre-commit install`
   - Explicar que los hooks corren golangci-lint y gofmt automáticamente
2. Actualizar sección de lint para mencionar `golangci-lint run ./...` además de `go vet`
3. Agregar sección "Commit Messages" explicando conventional commits:
   - Formato: `type(scope): description`
   - Tipos válidos: feat, fix, docs, test, refactor, ci, chore, perf, style
4. Mencionar `.editorconfig` en Code Conventions
5. Actualizar PR checklist para incluir: pre-commit pasa, golangci-lint pasa, conventional commit format

**Out**: Reescribir CONTRIBUTING.md desde cero, agregar screenshots o diagramas, documentar release process

## Estado inicial esperado

- `CONTRIBUTING.md` existe con secciones: Reporting Bugs, Suggesting Features, Development Setup, Code Conventions, Pull Requests
- No menciona pre-commit, golangci-lint, conventional commits, ni editorconfig

## Criterios de Aceptacion

- `CONTRIBUTING.md` contiene instrucciones de instalación de pre-commit (`pre-commit install`)
- `CONTRIBUTING.md` menciona `golangci-lint run ./...` en sección de lint
- `CONTRIBUTING.md` documenta formato de conventional commits con tipos válidos
- `CONTRIBUTING.md` menciona `.editorconfig`
- PR checklist incluye al menos: tests pasan, golangci-lint pasa, conventional commit format

## Fuente de verdad

- `CONTRIBUTING.md` — a modificar
- `.pre-commit-config.yaml` — referencia para instrucciones de hooks
- `.golangci.yml` — referencia para lint
