---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar analizador de frecuencia de campos y deteccion de tipos

**Story**: [S001 Schema Inference](README.md)

## Contexto

Para que `rootline init` pueda inferir un .stem schema, necesita un analizador que procese una lista de Records extraidos y calcule estadisticas: frecuencia de cada campo, valores unicos, y tipo inferido. Este paquete es la logica pura — el comando CLI (T002) lo consume.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/infer
interfaces:
  - nombre: Analyzer
    metodos:
      - nombre: Analyze
        input: "records []*extract.Record"
        output: "*InferredSchema"
      - nombre: FieldInfo
        input: ""
        output: "map[string]*FieldStats"
dependencias_externas: []
tests:
  - Campo en >80% de records se marca required
  - Campo con <20 valores unicos y >50% frecuencia se infiere como enum
  - Campo con valores variados se infiere como string
  - Lista de valores se infiere como list
  - 0 records retorna schema vacio
```

## Dependencias

- internal/extract (Record type)

## Alcance

**In**:
1. Struct `FieldStats` con: name, count, total_records, unique_values, inferred_type
2. Struct `InferredSchema` con: fields map, scope suggestion
3. Funcion `Analyze(records []*extract.Record) *InferredSchema`
4. Reglas de inferencia: required si presente en >80%, enum si <20 valores unicos y presencia >50%, string por default
5. Deteccion de tipo list (valores que son arrays YAML)

**Out**: Generacion de YAML (eso es T002), CLI command, inferencia de validate rules

## Estado inicial esperado

- internal/extract con Record type funcional
- Paquete internal/infer/ no existe (crear)

## Criterios de Aceptacion

- `Analyze(records)` con 10 records donde 9 tienen "estado" retorna campo required=true
- `Analyze(records)` con campo que tiene 3 valores unicos en 10 records retorna type=enum con values
- `Analyze(records)` con campo de valores variados retorna type=string
- `Analyze([]Record{})` retorna InferredSchema con fields vacio
- `go test ./internal/infer/ -race` pasa

## Fuente de verdad

- `internal/extract/extract.go` — Record struct
- `internal/rules/rules.go` — StemFile, SchemaField (target format)
