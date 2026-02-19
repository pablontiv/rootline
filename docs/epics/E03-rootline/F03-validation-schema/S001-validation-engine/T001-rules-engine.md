---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar 4 reglas de validacion

**Story**: [S001 Validation Engine](README.md)

## Contexto

El rules engine valida Record.Frontmatter contra el schema efectivo resuelto por el merge de .stem files. Las 4 reglas iniciales son estructurales: non_empty, enum, requires (condicional), exists. Reglas parametricas (format, max_length, pattern) estan diferidas.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: Validator
    metodos:
      - nombre: Validate
        input: "record *extract.Record, effective *StemFile"
        output: "[]ValidationError"
dependencias_externas: []
tests:
  - non_empty con campo vacio produce error
  - non_empty con campo presente pasa
  - enum con valor valido pasa
  - enum con valor invalido produce error listando valores validos
  - requires con condicion true y campo faltante produce error
  - requires con condicion false no chequea
  - exists con campo presente pasa
  - exists con campo ausente produce error
  - Campo required en schema faltante en frontmatter produce error
```

## Dependencias

- F02/S001 (StemFile con schema y validate sections)
- F02/S002 (Record con Frontmatter)

## Alcance

**In**:
1. Struct `ValidationError` con Rule, Field, Message, Source (path del .stem)
2. Funcion `Validate(record *extract.Record, effective *StemFile) []ValidationError`
3. Regla `non_empty`: campo existe y no es string vacio
4. Regla `enum`: valor esta en SchemaField.Values
5. Regla `requires`: si condicion matchea, campos listados deben existir
6. Regla `exists`: campo esta presente (aunque sea vacio)
7. Auto-check: campos con `required: true` en schema deben existir en frontmatter
8. Source tracking en cada error

**Out**: Parametric rules, custom validators, rule disabling

## Estado inicial esperado

- StemFile y Record disponibles
- Schema fields incluyen Type, Required, Values

## Criterios de Aceptacion

- Documento PRD con `Estado: "invalido"` produce enum error con valores validos listados
- Documento Task sin `estado` produce required field error
- Documento con `Estado: "Completado"` sin `Fecha` produce requires error (si regla definida)
- Cada ValidationError incluye source path del .stem que definio la regla
- Documento valido retorna lista vacia de errores

## Fuente de verdad

- `src/rootline/docs/research/I5-describe-contract.md` seccion 2.4 (validation rules table)
- `src/rootline/docs/research/I5-describe-contract.md` seccion 3 (real .stem examples con validate sections)
