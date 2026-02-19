---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar cobra command rootline validate

**Story**: [S002 Validate Command](README.md)

## Contexto

El validate command integra todo el pipeline: scanner descubre archivos, extractor produce Records, rules engine valida contra schema efectivo. El comando es el primer punto de contacto del usuario con Rootline.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline (cobra command)
interfaces:
  - nombre: validateCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: error
dependencias_externas:
  - github.com/spf13/cobra
tests:
  - validate con archivo valido retorna exit 0 y JSON con valid:true
  - validate con archivo invalido retorna exit 1 y JSON con errors
  - validate --all procesa todos los archivos en scope
  - validate con --field extrae dot-path del resultado
```

## Dependencias

- F03/S001 (rules engine + validation output)
- F02 completo (scanner, extractor, .stem merge)

## Alcance

**In**:
1. Cobra command `validate` con args: [file] o flag --all
2. Single file mode: resolve effective .stem, extract, validate, output JSON
3. All mode: scan directory, validate each Record, output batch JSON
4. `--field` flag para dot-path extraction del resultado
5. Exit code: 0 = all valid, 1 = any error
6. Integration tests con fixtures

**Out**: Table output format (futuro), watch mode

## Estado inicial esperado

- Cobra skeleton con validate stub (F01/S001/T002)
- Scanner, Extractor, Rules Engine funcionales

## Criterios de Aceptacion

- `rootline validate test.md` produce JSON ValidationResult
- `rootline validate --all` produce JSON BatchValidationResult
- Exit code 0 para archivos validos
- Exit code 1 para archivos con errores de validacion
- `rootline validate test.md --field errors` extrae solo la lista de errores

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` seccion 3 (Commands: validate)
