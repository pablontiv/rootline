---
tipo: historia
cliente: Platform Owner
---
# S002: Advanced Test Coverage

**Feature**: [F02 CI Quality Gates](../README.md)
**Capacidad**: Paths críticos tienen fuzz tests para descubrir crashes y benchmarks para establecer línea base de performance

## Antes / Despues

**Antes**: No hay fuzz tests pese a que YAML parsing es un target natural (input no confiable). No hay benchmarks — no se puede medir impacto de performance de cambios en WalkUp, Scan o Execute.

**Despues**: `internal/extract/` tiene fuzz tests que se ejecutan con `go test -fuzz`. Hot paths (WalkUp, Scan, Execute) tienen benchmarks ejecutables con `go test -bench`. CI puede correr fuzz tests con tiempo limitado.

## Criterios de Aceptacion (semanticos)

- [ ] `go test ./internal/extract/ -fuzz FuzzExtract -fuzztime 30s` corre sin crashes
- [ ] `go test ./internal/rules/ -bench BenchmarkWalkUp` produce output de benchmark
- [ ] `go test ./internal/index/ -bench BenchmarkScan` produce output de benchmark
- [ ] `go test ./internal/query/ -bench BenchmarkExecute` produce output de benchmark

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-fuzz-tests-yaml-parsing.md) | Agregar fuzz tests para YAML frontmatter parsing |
| [T002](T002-benchmarks-hot-paths.md) | Agregar benchmarks para WalkUp, Scan y Execute |

## Fuente de verdad

- `internal/extract/extract.go` — MarkdownExtractor.Extract, target para fuzz
- `internal/rules/discovery.go` — WalkUp, target para benchmark
- `internal/index/scan.go` — Scan, target para benchmark
- `internal/query/query.go` — Execute, target para benchmark
