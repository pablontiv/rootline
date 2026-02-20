# S001: Directory-wide Fix

**Feature**: [F06 E05 Hardening](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: fix opera sobre directorios completos con batch output, igual que validate --all

## Antes / Despues

**Antes**: `rootline fix` solo acepta archivos individuales. Para reparar un directorio completo, el usuario debe listar manualmente cada archivo con errores. `rootline validate --all` detecta errores en batch pero `fix` no puede repararlos en batch.

**Despues**: `rootline fix --all` escanea todos los archivos del directorio, aplica reparaciones, y produce un resumen batch (JSON con summary y tabla con columnas File/Fixed/Changes). El pipeline validate→fix funciona end-to-end sin friccion.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline fix --all` repara todos los archivos con errores en un directorio
- [ ] Output JSON tiene estructura batch con summary (total/fixed/skipped)
- [ ] Output table muestra columnas File, Fixed, Changes
- [ ] `rootline fix --all --dry-run` muestra cambios sin modificar archivos

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-fix-all-flag.md) | Agregar --all flag a fix con scan de directorio |
| [T002](T002-fix-batch-output.md) | Batch JSON/table output con summary |

## Fuente de verdad

- `cmd/rootline/fix.go` — fix command actual
- `cmd/rootline/validate.go` — referencia para --all pattern (runValidateAll)
- `internal/rules/validate_result.go` — BatchValidationResult pattern
