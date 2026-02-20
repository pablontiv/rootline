---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar computeNextSequence y wiring en NewDescribeResult

**Story**: [S001 Core Engine](README.md)

## Contexto

Una vez que SchemaField tiene Prefix/Digits/Next (T001), hay que implementar la logica que computa el valor de Next. La funcion `computeNextSequence(dirPath string, field SchemaField) string` escanea el directorio con `os.ReadDir`, filtra archivos con regex `^{prefix}(\d+)`, encuentra el maximo numerico, y retorna `prefix + fmt.Sprintf("%0{digits}d", max+1)`. Esta funcion se llama desde `NewDescribeResult` para cada campo con `type == "sequence"`.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: computeNextSequence
    metodos:
      - nombre: computeNextSequence
        input: "dirPath string, field SchemaField"
        output: "string (ej: 'T004')"
  - nombre: NewDescribeResult (modificar)
    metodos:
      - nombre: NewDescribeResult
        input: "path string, entries []StemEntry, effective *StemFile"
        output: "*DescribeResult (con Next poblado en campos sequence)"
dependencias_externas:
  - os (ReadDir)
  - regexp
  - strconv
  - fmt
tests:
  - directorio vacio -> "T001"
  - directorio con T001,T002 -> "T003"
  - archivos no-matching ignorados (README.md, .stem)
  - prefix E digits 2: E01,E02 -> "E03"
```

## Alcance

**In**:
1. Implementar `computeNextSequence(dirPath string, field SchemaField) string` en describe.go
2. En `NewDescribeResult`, iterar `effective.Schema`, para cada campo con `Type == "sequence"` llamar `computeNextSequence` y asignar `field.Next`
3. El `dirPath` se obtiene del `path` argument de `NewDescribeResult` (ya es relativo/absoluto segun el caller)

**Out**: Cambios a SchemaField struct (T001), tests (T003), .stem files (S002/T001)

## Estado inicial esperado

- T001 completado: SchemaField tiene Prefix, Digits, Next
- `internal/rules/describe.go` contiene `NewDescribeResult` en linea ~24

## Criterios de Aceptacion

- `computeNextSequence` retorna "T001" para directorio vacio con prefix=T digits=3
- `computeNextSequence` retorna "T003" si T001-a.md y T002-b.md existen
- README.md, .stem, y otros archivos sin el patron son ignorados
- `go test ./internal/rules/ -race` pasa
- `rootline describe <dir-con-sequence-stem> --field schema.id.next` retorna el valor correcto

## Fuente de verdad

- `internal/rules/describe.go` — NewDescribeResult a modificar, computeNextSequence a agregar
- `internal/rules/rules.go` — SchemaField.Type constante "sequence"
