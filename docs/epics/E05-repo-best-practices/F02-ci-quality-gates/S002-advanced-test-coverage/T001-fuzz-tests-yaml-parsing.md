---
estado: Completed
tipo: software-test
ejecutable_en: 1 sesion
---
# T001: Agregar fuzz tests para YAML frontmatter parsing

**Story**: [S002 Advanced Test Coverage](README.md)

## Contexto

`internal/extract/` es el punto de entrada para input no confiable — recibe archivos markdown arbitrarios y parsea YAML frontmatter. Fuzz testing genera inputs aleatorios automáticamente para descubrir panics, infinite loops, y memory corruption que los tests manuales no cubren. Go tiene soporte nativo de fuzzing desde 1.18 con `testing.F`.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
paquete: internal/extract
cobertura_objetivo: n/a (fuzz, no line coverage)
tipos_test:
  - unit (fuzz)
fixtures:
  - Corpus seeds generados a partir de archivos .md existentes en testdata/ y docs/
```

## Dependencias

- Ninguna

## Alcance

**In**:
1. Crear `internal/extract/fuzz_test.go`
2. Implementar `FuzzExtract(f *testing.F)` que:
   - Agrega seeds del corpus: frontmatter válido, frontmatter vacío, YAML inválido, markdown sin frontmatter, archivo vacío
   - Fuzzea llamando a `MarkdownExtractor{}.Extract(path)` con archivos generados aleatoriamente
   - Verifica que no hay panic (fuzz default) y que el resultado es Record o error (no nil, nil)
3. Agregar al menos 5 seed corpus entries basadas en patrones reales del proyecto

**Out**: Fuzz tests para otros packages, integrar fuzz en CI con tiempo ilimitado, modificar el extractor

## Estado inicial esperado

- `internal/extract/extract.go` tiene `MarkdownExtractor` con método `Extract(path string) (Record, error)`
- `internal/extract/extract_test.go` tiene tests unitarios convencionales
- No existe `fuzz_test.go`

## Criterios de Aceptacion

- `go test ./internal/extract/ -fuzz FuzzExtract -fuzztime 30s` corre sin crashes
- `internal/extract/fuzz_test.go` existe con al menos 5 seed corpus entries
- `go test ./internal/extract/ -run FuzzExtract` pasa (seeds como test cases normales)
- Si el fuzzer encuentra un crash, crear test case de regresión antes de cerrar el task

## Fuente de verdad

- `internal/extract/extract.go` — MarkdownExtractor.Extract
- `internal/extract/extract_test.go` — tests existentes como referencia
- `internal/extract/registry.go` — Registry pattern para entender el API
