---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Add path argument to validate --all

**Story**: [S001 Validate and Fix Path Support](README.md)
**Contribuye a**: `rootline validate --all docs/epics/` escanea solo docs/epics

## Preserva

- INV1: Sin argumentos, `--all` sigue usando CWD
  - Verificar: `rootline validate --all --output json | python3 -c "import json,sys; print(json.load(sys.stdin)['summary']['total'] > 0)"`

## Contexto

`validate --all` usa `filepath.Abs(".")` hardcoded como scan root (línea 123 de validate.go). Los otros 4 comandos transversales (`tree`, `query`, `stats`, `graph`) usan `args[0]` cuando se proporciona. El patrón canónico está en `tree.go:49-55`:

```go
scanRoot := "."
if len(args) > 0 {
    scanRoot = args[0]
}
absRoot, err := filepath.Abs(scanRoot)
```

## Alcance

**In**:
1. Cambiar firma de `runValidateAll(cmd *cobra.Command)` a `runValidateAll(cmd *cobra.Command, args []string)`
2. Reemplazar `filepath.Abs(".")` con patrón `scanRoot` de tree.go
3. Actualizar llamada en `runValidate` para pasar `args`

**Out**: No modificar `runValidateStaged` (usa git staging, no directorio)

## Estado inicial esperado

- `cmd/rootline/validate.go` existe con `runValidateAll` que usa `filepath.Abs(".")`
- `cmd/rootline/tree.go` tiene el patrón de referencia en líneas 49-55

## Criterios de Aceptacion

- `go build ./cmd/rootline/` compila sin errores
- `go test ./... -race` pasa
- `rootline validate --all docs/epics/ --output json | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['summary']['total'])"` retorna count > 0
- `rootline validate --all --output json | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['summary']['total'])"` sigue funcionando (backward compatible)

## Fuente de verdad

- `cmd/rootline/validate.go` — líneas 49-50 (llamada), 120-123 (función)
- `cmd/rootline/tree.go` — líneas 49-55 (patrón de referencia)
