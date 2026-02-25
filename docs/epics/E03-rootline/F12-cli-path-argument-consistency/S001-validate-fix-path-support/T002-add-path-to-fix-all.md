---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Add path argument to fix --all

**Story**: [S001 Validate and Fix Path Support](README.md)
**Contribuye a**: `rootline fix --all docs/epics/` escanea solo docs/epics

## Preserva

- INV2: Sin argumentos, `--all` sigue usando CWD
  - Verificar: `rootline fix --all --dry-run --output json | python3 -c "import json,sys; print('ok')"`

## Contexto

`fix --all` usa `filepath.Abs(".")` hardcoded como scan root (línea 153 de fix.go). Mismo bug que validate — copió el patrón original sin `args[0]`. El patrón canónico está en `tree.go:49-55`.

## Alcance

**In**:
1. Cambiar firma de `runFixAll(ctx, cmd)` a `runFixAll(ctx, cmd, args)`
2. Reemplazar `filepath.Abs(".")` con patrón `scanRoot` de tree.go
3. Actualizar llamada en `runFix` para pasar `args`

**Out**: No modificar el modo single-file de fix (ya acepta paths correctamente)

## Estado inicial esperado

- `cmd/rootline/fix.go` existe con `runFixAll` que usa `filepath.Abs(".")`
- T001 completado (validate ya tiene el fix)

## Criterios de Aceptacion

- `go build ./cmd/rootline/` compila sin errores
- `go test ./... -race` pasa
- `rootline fix --all docs/epics/ --dry-run --output json` ejecuta sin error
- `rootline fix --all --dry-run --output json` sigue funcionando (backward compatible)

## Fuente de verdad

- `cmd/rootline/fix.go` — líneas 73 (llamada), 152-153 (función)
- `cmd/rootline/tree.go` — líneas 49-55 (patrón de referencia)
