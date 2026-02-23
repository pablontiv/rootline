---
estado: Specified
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Update test files and testdata stems

**Story**: [S002 Code & Tooling Alignment](README.md)

[[blocks:T001-update-production-code]]

## Contexto

~209 ocurrencias de `"Completado"` y `"Bloqueada"` existen en 22 test files y 3 testdata .stem files. Los tests son self-contained (definen sus propios schemas inline) y seguirian pasando con valores viejos, pero para consistencia y claridad deben alinearse con el nuevo modelo en ingles.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
paquete: internal/ (multiple packages)
cobertura_objetivo: mantener >= 85%
tipos_test:
  - unit
  - integration
  - e2e
fixtures:
  - internal/rules/testdata/epics.stem
  - internal/rules/testdata/task.stem
  - internal/rules/testdata/prd.stem
```

## Alcance

**In**:
1. Bulk replace `"Completado"` → `"Completed"` en todos los `*_test.go` bajo `internal/`
2. Bulk replace `"Bloqueada"` → `"Blocked"` en todos los `*_test.go` bajo `internal/`
3. Update `internal/rules/testdata/epics.stem`, `task.stem`, `prd.stem` enum values
4. Verificar que `go test ./... -race` pasa

**Out**: Production code (T001), skill files (T003)

## Estado inicial esperado

- T001 completado (production code actualizado)
- ~209 ocurrencias de valores viejos en test files
- 3 testdata .stem files con enum values en espanol

## Criterios de Aceptacion

- `grep -r "Completado" internal/ --include="*.go" | wc -l` retorna 0
- `grep -r "Bloqueada" internal/ --include="*.go" | wc -l` retorna 0
- `grep "Completado" internal/rules/testdata/*.stem | wc -l` retorna 0
- `go test ./... -race` pasa sin errores
- Coverage >= 85%

## Fuente de verdad

- `internal/**/*_test.go` (22 files)
- `internal/rules/testdata/*.stem` (3 files)
