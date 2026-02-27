---
estado: Completed
tipo: historia
---
# S004: Validate & Doctor Unification

**Feature**: [F11 Aggregate Auto-Generation & Command Consolidation](../README.md)
**Capacidad**: `rootline validate` realiza todos los health checks de stem + documentos; `doctor` queda como alias deprecado
**Cubre**: Unificación de comandos de health-check en modelo validate + fix

## Antes / Despues

**Antes**: 3 comandos con overlap: `validate` (documentos vs schema), `doctor` (7 checks de salud del .stem), `fix` (proposals + escritura). Doctor y validate son ambos read-only pero con scope diferente. Usuarios no saben cuál usar.

**Despues**: `validate` ejecuta los 7 checks de doctor + validación de documentos en un solo comando. `doctor` es alias con deprecation warning. Modelo estándar: validate = check todo (CI gate), fix = corregir (proposals + escritura).

## Criterios de Aceptacion (semanticos)

- [ ] `rootline validate dir/` ejecuta los 7 checks de stem health (scope-match, type-consistency, enum-values, etc.)
- [ ] `rootline doctor dir/` emite "Warning: 'doctor' is deprecated, use 'validate' instead" y ejecuta validate
- [ ] Tests de doctor migrados a validate_test.go pasan

## Invariantes

- INV1: Tests existentes siguen pasando
  - Verificar: `go test ./cmd/rootline/ -run "TestValidate|TestDoctor" -v`

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-unify-validate-doctor.md) | Migrar 7 checks de doctor a validate, deprecar doctor como alias |

## Fuente de verdad

- `cmd/rootline/doctor.go` — 7 checks existentes
- `cmd/rootline/validate.go` — runValidate
- `cmd/rootline/doctor_test.go` — tests existentes
