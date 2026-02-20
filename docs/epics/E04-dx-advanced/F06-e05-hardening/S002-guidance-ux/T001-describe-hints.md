---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: describe sugiere init cuando no hay .stem

**Story**: [S002 Guidance UX](README.md)

## Contexto

Cuando `rootline describe` se ejecuta en un directorio sin .stem files, retorna schema vacio sin explicacion. El usuario no sabe que debe correr `rootline init` para generar un schema. E05 T003 confirmo este comportamiento.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline + internal/rules
interfaces:
  - nombre: DescribeResult
    metodos:
      - nombre: Hints field
        input: "[]string"
        output: "JSON serializable"
dependencias_externas: []
tests:
  - describe sin .stem incluye hint "Run rootline init" en JSON
  - describe sin .stem muestra mensaje hint en table output
  - describe con .stem no incluye hints
```

## Dependencias

- describe command existente (`cmd/rootline/describe.go`)
- DescribeResult struct (`internal/rules/describe.go`)

## Alcance

**In**:
1. Agregar campo `Hints []string` a `DescribeResult` en `internal/rules/describe.go`
2. En `runDescribe()`: detectar cuando `effective == nil || len(effective.Schema) == 0`
3. Agregar hint: `"No .stem schema found. Run 'rootline init <path>' to infer schema from existing files."`
4. En `renderDescribeTable()`: si hay hints, mostrarlos como nota al final
5. Tests

**Out**: Sugerir campos especificos, auto-ejecutar init, cambiar exit code

## Estado inicial esperado

- `cmd/rootline/describe.go` con runDescribe funcional
- `internal/rules/describe.go` con DescribeResult struct (sin campo Hints)

## Criterios de Aceptacion

- `rootline describe /tmp/no-stem/ --output json` contiene `"hints":["No .stem schema found..."]`
- `rootline describe /tmp/no-stem/ --output table` muestra mensaje de hint
- `rootline describe /tmp/with-stem/` no contiene hints (null o vacio)
- `go test ./... -race` pasa

## Fuente de verdad

- `cmd/rootline/describe.go` — describe command
- `internal/rules/describe.go` — DescribeResult struct
- `cmd/rootline/doctor.go:83-88` — referencia para warning pattern
