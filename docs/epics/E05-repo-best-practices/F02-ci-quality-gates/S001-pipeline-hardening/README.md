---
estado: Specified
tipo: historia
cliente: Platform Owner
---
# S001: Pipeline Hardening

**Feature**: [F02 CI Quality Gates](../README.md)
**Capacidad**: CI tiene gates reproducibles que detectan drift en lint, dependencias y cobertura

## Antes / Despues

**Antes**: golangci-lint usa `version: latest` en CI (no reproducible, puede romper entre runs). No hay verificación de `go mod tidy` (go.sum puede estar dirty sin que CI lo detecte). Coverage se genera como artefacto pero no se reporta ni enforcea — puede degradar silenciosamente.

**Despues**: golangci-lint está pinneado a la misma versión que pre-commit (`v2.1.6`). CI falla si `go mod tidy` produce diff. Coverage tiene threshold mínimo y CI falla si baja.

## Criterios de Aceptacion (semanticos)

- [ ] golangci-lint en CI usa versión explícita, no `latest`
- [ ] CI falla si `go mod tidy` produce cambios en go.mod o go.sum
- [ ] CI falla si coverage baja del threshold definido

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-pin-golangci-lint-version.md) | Pinear golangci-lint a versión explícita en CI |
| [T002](T002-add-go-mod-tidy-check.md) | Agregar verificación de go mod tidy en CI |
| [T003](T003-integrate-coverage-threshold.md) | Integrar reporte de coverage con threshold mínimo |

## Fuente de verdad

- `.github/workflows/ci.yml` — pipeline a modificar
- `.golangci.yml` — referencia de linter version
- `.pre-commit-config.yaml` — versión pinneada de referencia
