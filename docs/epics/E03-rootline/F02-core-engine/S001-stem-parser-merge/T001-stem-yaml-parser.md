---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Parsear archivos .stem YAML a structs Go

**Story**: [S001 Stem Parser and Merge](README.md)

## Contexto

Los archivos `.stem` son YAML con secciones definidas: version, scope, schema, validate, derive, state, links. Este Task implementa el parser que convierte un archivo .stem en una estructura Go tipada. El parser no hace merge ni discovery — solo parsea un archivo individual.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/rules
interfaces:
  - nombre: StemFile
    metodos:
      - nombre: ParseStem
        input: "path string, content []byte"
        output: "*StemFile, error"
dependencias_externas:
  - gopkg.in/yaml.v3
tests:
  - Parsea .stem con todas las secciones
  - Parsea .stem con solo version y scope
  - Error en YAML malformado
  - Secciones desconocidas se ignoran (forward compat)
```

## Dependencias

- F01/S001 completado (Go module con paquete internal/rules/)

## Alcance

**In**:
1. Definir struct `StemFile` con campos: Version, Scope, Schema, Validate, Derive, State, Links
2. `Schema` es `map[string]SchemaField` donde SchemaField tiene Type, Required, Values, Default, Source
3. `Validate` es `[]ValidationRule` con Rule, If, Then, Source
4. `Scope` tiene Match (glob pattern)
5. Funcion `ParseStem(path string, content []byte) (*StemFile, error)`
6. Tests unitarios con fixtures .stem reales del I5 research

**Out**: Walk-up discovery (T002), merge (T003), derive/state/links processing (deferred)

## Estado inicial esperado

- Go module compilable
- Paquete `internal/rules/` existe

## Criterios de Aceptacion

- `ParseStem` parsea correctamente los 5 ejemplos de I5 (docs/.stem, prd/.stem, epics/.stem, S001/.stem, research/.stem)
- `ParseStem` retorna error para YAML malformado
- `ParseStem` ignora secciones desconocidas sin error
- `StemFile.Schema` es accesible como map con SchemaField tipados
- Tests cubren >90% del paquete

## Fuente de verdad

- `src/rootline/docs/research/I5-describe-contract.md` seccion 3 (Real .stem Examples)
- `src/rootline/README.md` seccion "Example .stem"
