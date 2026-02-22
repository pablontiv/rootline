---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Agregar benchmarks para WalkUp, Scan y Execute

**Story**: [S002 Advanced Test Coverage](README.md)

## Contexto

Los tres hot paths de rootline son: `WalkUp` (descubrimiento de .stem files hacia arriba), `Scan` (escaneo de directorio con .stemignore), y `Execute` (evaluación de queries con expr). Sin benchmarks no se puede medir el impacto de performance de cambios futuros ni detectar regresiones. Go tiene soporte nativo de benchmarking con `testing.B`.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
paquete: internal/rules, internal/index, internal/query
cobertura_objetivo: n/a (benchmark, no line coverage)
tipos_test:
  - unit (benchmark)
fixtures:
  - Árbol de directorios temporal con profundidad variable para WalkUp
  - Directorio con N archivos markdown para Scan
  - Set de records con queries variadas para Execute
```

## Dependencias

- Ninguna

## Alcance

**In**:
1. Crear `internal/rules/bench_test.go` con `BenchmarkWalkUp` — árbol de 10 niveles de profundidad con .stem en cada nivel
2. Crear `internal/index/bench_test.go` con `BenchmarkScan` — directorio con 100 archivos .md
3. Crear `internal/query/bench_test.go` con `BenchmarkExecute` — 50 records con query `eq`, `contains`, y `and` compuesto
4. Usar `b.ResetTimer()` después del setup del fixture
5. Usar `t.TempDir()` para aislamiento de fixtures

**Out**: Benchmark regression en CI (solo establecer baseline), benchmarks de extract (cubierto por fuzz), optimización de código

## Estado inicial esperado

- `internal/rules/discovery.go` exporta `WalkUp(target, boundary string) ([]StemFile, error)`
- `internal/index/scan.go` exporta `Scan(root string) ([]FileInfo, error)` o similar
- `internal/query/query.go` exporta `Execute(records, conditions)` o similar
- No existen archivos bench_test.go

## Criterios de Aceptacion

- `go test ./internal/rules/ -bench BenchmarkWalkUp -benchmem` produce output con ns/op y allocs/op
- `go test ./internal/index/ -bench BenchmarkScan -benchmem` produce output con ns/op y allocs/op
- `go test ./internal/query/ -bench BenchmarkExecute -benchmem` produce output con ns/op y allocs/op
- `go test ./... -race` pasa sin regresiones (benchmarks no corren con -race por defecto, pero test mode sí)

## Fuente de verdad

- `internal/rules/discovery.go` — WalkUp signature y params
- `internal/index/scan.go` — Scan signature y params
- `internal/query/query.go` — Execute signature y params
- `internal/rules/rules_test.go` — patrón de setup de fixtures con t.TempDir
