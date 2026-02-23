---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Agregar Prefix/Digits/Next a SchemaField struct

**Story**: [S001 Core Engine](README.md)

## Contexto

`SchemaField` en `internal/rules/rules.go:88` define los campos de un campo de schema. Para soportar `type: sequence`, necesita tres campos nuevos: `Prefix` (string leido del YAML, ej: "T"), `Digits` (int leido del YAML, ej: 3), y `Next` (string computado en runtime, nunca del YAML). El campo `Next` debe incluirse en el JSON output para que `--field schema.id.next` funcione via el mecanismo de dot-path extraction existente.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: SchemaField
    metodos:
      - nombre: (struct fields)
        input: yaml tags para Prefix y Digits; yaml:"-" para Next
        output: json tags con omitempty para los tres
dependencias_externas: []
tests:
  - SchemaField con type sequence se deserializa correctamente del YAML
  - SchemaField.Next se omite en JSON cuando vacio
  - SchemaField.Next aparece en JSON cuando tiene valor
```

## Alcance

**In**:
1. Agregar `Prefix string \`yaml:"prefix" json:"prefix,omitempty"\`` a SchemaField
2. Agregar `Digits int \`yaml:"digits" json:"digits,omitempty"\`` a SchemaField
3. Agregar `Next string \`yaml:"-" json:"next,omitempty"\`` a SchemaField
4. Verificar que merge de SchemaField no rompe con los nuevos campos (mapas se mergean por clave)

**Out**: Implementacion de computeNextSequence (T002), cambios a otros structs

## Estado inicial esperado

- `internal/rules/rules.go` existe con SchemaField definido en linea ~88
- Los tests existentes de rules pasan: `go test ./internal/rules/ -race`

## Criterios de Aceptacion

- `go build ./...` pasa sin errores
- `go test ./internal/rules/ -race` pasa (tests existentes no se rompen)
- Un .stem YAML con `id: {type: sequence, prefix: T, digits: 3}` se deserializa con SchemaField{Prefix:"T", Digits:3}
- El JSON output de SchemaField incluye `"prefix":"T","digits":3` pero NO incluye `"next"` cuando Next == ""

## Fuente de verdad

- `internal/rules/rules.go:88` — SchemaField struct a modificar
- `internal/rules/merge_test.go` — tests de merge a verificar que siguen pasando
