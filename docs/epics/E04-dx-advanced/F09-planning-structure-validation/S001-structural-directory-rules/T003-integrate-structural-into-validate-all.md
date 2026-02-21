---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Integrar structural checks en validate --all pipeline

**Story**: [S001 Structural Directory Rules](README.md)

[[blocks:T002-implement-validate-directory]]

## Contexto

Con T001 y T002 completados, existe `ValidateDirectory(dir, stem)` que retorna `[]ValidationError`. Ahora se necesita integrarlo en el comando `rootline validate --all` para que al escanear un directorio, tambien ejecute validaciones estructurales en cada directorio que tiene un `.stem` con bloque `structural`.

El path `--all` en `cmd/rootline/validate.go` usa `index.Scan` con `ScopeResolver` para encontrar archivos, y luego valida cada uno. La integracion agrega un paso adicional: para cada directorio visitado con un `.stem` que tiene `structural:`, ejecutar `ValidateDirectory`.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: runValidateAll (modificar existente)
    metodos:
      - nombre: runValidateAll
        input: ""
        output: "*rules.BatchValidationResult, error"
dependencias_externas: []
tests:
  - validate --all con .stem structural reporta directorios sin README
  - validate --all sin structural rules no cambia comportamiento
  - Resultados estructurales tienen Path con trailing slash (dir/)
  - validate --all --strict con structural warnings retorna exit 1
  - JSON output incluye resultados estructurales en Results array
```

## Dependencias

- T002 completado (ValidateDirectory disponible)

## Alcance

**In**:
1. En `runValidateAll`, despues del scan de archivos, recolectar directorios unicos visitados
2. Para cada directorio, obtener effective .stem via WalkUp + merge
3. Si effective .stem tiene `Structural.Subdirs` no-zero, llamar `ValidateDirectory(dir, stem)`
4. Convertir resultados en `ValidationResult` con `Path = relDir + "/"` (trailing slash)
5. Incluir en `BatchValidationResult.Results`
6. Tests de integracion CLI

**Out**: No modificar otros paths del validate (single file, --staged). No agregar flag especifico para structural.

## Estado inicial esperado

- `cmd/rootline/validate.go` tiene `runValidateAll` que solo valida archivos
- `ValidateDirectory` existe en `internal/rules/structural.go` (de T002)

## Criterios de Aceptacion

- `rootline validate --all` en directorio con .stem structural reporta directorios invalidos
- Path de resultados estructurales termina en `/` (ej: `docs/epics/E03-rootline/`)
- `rootline validate --all --output json` incluye resultados estructurales en `results` array
- `rootline validate --all --strict` con warnings estructurales retorna exit code 1
- `rootline validate --all` sin .stem structural se comporta identico al baseline
- `go test ./cmd/rootline/ -run TestValidateStructural` pasa

## Fuente de verdad

- `cmd/rootline/validate.go` — runValidateAll function
- `internal/rules/structural.go` — ValidateDirectory (de T002)
- `cmd/rootline/commands_test.go` — patron de tests CLI
