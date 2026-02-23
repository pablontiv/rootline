---
tipo: feature
---
# F02: CI Quality Gates

**Epic**: [E05](../README.md)
**Objetivo**: El pipeline CI detecta regresiones de calidad con gates reproducibles: lint versionado, go mod tidy, coverage threshold, fuzz tests y benchmarks
**Beneficio**: Previene drift silencioso en dependencias, regresiones de cobertura, y establece línea base de performance
**Milestone**: CI falla si: lint version difiere, go.mod está dirty, coverage baja del threshold, o fuzz tests encuentran crash

## Scope

**In**: Pinear golangci-lint en CI, verificar go mod tidy, integrar coverage con threshold, agregar fuzz tests para YAML parsing, agregar benchmarks para hot paths
**Out**: Integración con Codecov/Coveralls externo, benchmark regression CI gate (solo baseline)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-pipeline-hardening/) | Pipeline Hardening | CI tiene gates reproducibles para lint, mod tidy y coverage |
| [S002](S002-advanced-test-coverage/) | Advanced Test Coverage | Fuzz tests y benchmarks cubren paths críticos |

## Dependencias

- F01 parcial — gosec habilitado en F01/T002 se aprovecha aquí

## Fuente de verdad

- `.github/workflows/ci.yml` — pipeline a modificar
- `.golangci.yml` — configuración de linters
- `internal/extract/` — target principal para fuzz tests
- `internal/rules/discovery.go` — WalkUp, target para benchmarks
- `internal/index/scan.go` — Scan, target para benchmarks
- `internal/query/` — Execute, target para benchmarks
