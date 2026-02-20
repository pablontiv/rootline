---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Agregar --all flag a fix con scan de directorio

**Story**: [S001 Directory-wide Fix](README.md)

## Contexto

El comando `fix` solo acepta archivos individuales (`fix <file> [files...]`). El spec original de E04/F02/S002/T002 contemplaba `fix --all` pero no se implemento. `validate` ya tiene `--all` con `runValidateAll()` que usa `index.Scan` — el patron es identico para fix.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: fixCmd
    metodos:
      - nombre: runFixAll
        input: "cmd *cobra.Command"
        output: "error"
dependencias_externas: []
tests:
  - fix --all en directorio con .stem y errores repara todos los archivos
  - fix --all sin .stem no modifica nada
  - fix --all --dry-run muestra cambios sin escribir
  - fix --all sin argumentos extra funciona
```

## Dependencias

- fix command existente (`cmd/rootline/fix.go`)
- validate --all pattern (`cmd/rootline/validate.go:115-153`)

## Alcance

**In**:
1. Agregar `fixAll bool` flag al fixCmd
2. Cambiar Args de `cobra.MinimumNArgs(1)` a custom validation (requiere args OR --all)
3. Implementar `runFixAll()` modelando `runValidateAll()`:
   - `index.Scan(root, reg, index.WithScopeResolver(resolver))`
   - Para cada record: WalkUp, MergeStemFiles, Validate, applyFixes si hay errores
4. Agregar `fixAll` a `resetFlags()` en commands_test.go
5. Tests en fix_test.go

**Out**: Batch output (T002), recursive subdirectory scan, interactive confirmation

## Estado inicial esperado

- `cmd/rootline/fix.go` con runFix, applyFixes, closestMatch, rewriteFrontmatter
- `cmd/rootline/validate.go` con runValidateAll como referencia
- `cmd/rootline/commands_test.go` con resetFlags pattern

## Criterios de Aceptacion

- `rootline fix --all` en directorio con .stem repara archivos con errores
- `rootline fix --all --dry-run` muestra cambios sin modificar
- `rootline fix --all` sin .stem reporta 0 fixes
- `rootline fix` sin args ni --all retorna error de uso
- `go test ./cmd/rootline/ -race -run TestFix` pasa
- `go test ./... -race` pasa

## Fuente de verdad

- `cmd/rootline/fix.go` — archivo a modificar
- `cmd/rootline/validate.go:115-153` — runValidateAll pattern
- `cmd/rootline/commands_test.go:39-56` — resetFlags
