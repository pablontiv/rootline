---
estado: Diferida
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Evaluar expresiones de .stem derive contra records

**Story**: [S002 Derivation Pipeline](README.md)

## Contexto

StemFile.Derive es actualmente `map[string]any`. Necesita tipificarse para contener expresiones string que se evaluan contra records. El evaluador toma el .stem efectivo (merged), extrae las expresiones de derive, y las evalua con el Evaluator de S001, poblando `Record.Derived`.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/derive
interfaces:
  - nombre: DeriveFields
    metodos:
      - nombre: DeriveRecord
        input: "record *extract.Record, effective *rules.StemFile, children []*extract.Record"
        output: "map[string]any, error"
dependencias_externas: []
tests:
  - DeriveRecord con derive slug=slugify(titulo) retorna {slug: valor}
  - DeriveRecord con expresion referenciando children funciona
  - DeriveRecord sin derive en .stem retorna map vacio
  - DeriveRecord con expresion invalida retorna error con contexto
```

## Dependencias

- S001 (Evaluator + builtins)
- internal/rules (StemFile with Derive)
- internal/extract (Record)

## Alcance

**In**:
1. Tipificar StemFile.Derive: map[string]string (field name → expression string)
2. O mantener map[string]any y extraer expression strings en runtime
3. Funcion `DeriveRecord(record, effective, children)` que evalua todas las expresiones
4. Agregar campo `Derived map[string]any` a extract.Record
5. Environment para eval: frontmatter fields + "children" variable
6. Error handling: expresion invalida no bloquea derivacion de otros campos

**Out**: Caching de compilacion, derivacion en cascade (campo derivado usa otro campo derivado), derived field validation

## Estado inicial esperado

- internal/derive/ con Evaluator y builtins (S001)
- StemFile.Derive parsea como map[string]any
- Record struct en internal/extract/

## Criterios de Aceptacion

- `DeriveRecord(record, stem, nil)` con derive:{slug: "slugify(titulo)"} retorna map con slug
- `DeriveRecord(record, stem, children)` con derive:{total: "len(children)"} retorna conteo
- `DeriveRecord(record, stem_sin_derive, nil)` retorna map vacio
- Record.Derived se puebla correctamente
- `go test ./internal/derive/ -race` pasa

## Fuente de verdad

- `internal/derive/derive.go` — Evaluator (S001/T001)
- `internal/rules/rules.go` — StemFile.Derive
- `internal/extract/extract.go` — Record struct
