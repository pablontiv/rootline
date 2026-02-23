---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Wire RecordResolver y LinkRule.Field en derive pipeline

**Story**: [S001 Graph Field Resolution](README.md)

## Contexto

El derive engine (`internal/derive/`) evalua expresiones por record, pero solo tiene acceso al propio record y sus children (via aggregate). Los wiki-links (`[[blocks:T001-name]]`) estan en `record.Links` como slice de `extract.Link{Type, Target}`. La regla `LinkRule` en el .stem tiene un campo `Field` que se parsea pero nunca se consume. Esta task wirea todo: construye un RecordResolver en DeriveAll, lo pasa a DeriveRecord, y inyecta valores de campos enlazados en el env de derive.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/derive
interfaces:
  - nombre: RecordResolver
    metodos:
      - nombre: Resolve
        input: "path string"
        output: "*extract.Record"
dependencias_externas: []
tests:
  - Record con [[blocks:B]] donde B tiene estado Completado → env["blocked_by"] = ["Completado"]
  - Record sin links → env sin variables de link
  - Record con link a target inexistente → skip graceful (no error)
  - Multiples links del mismo tipo → slice con todos los valores
```

## Dependencias

- internal/derive/ existente (DeriveAll, DeriveRecord)
- internal/rules/ existente (LinkRule.Field)
- internal/extract/ existente (Record.Links)

## Alcance

**In**:
1. Nuevo archivo `internal/derive/links.go` con RecordResolver type
2. En DeriveAll (`pipeline.go`): construir `map[string]*Record` lookup del record set completo
3. Pasar RecordResolver a DeriveRecord
4. En DeriveRecord (`record.go`): iterar `record.Links`, buscar LinkRule por tipo, si `rule.Field != ""` → resolver record → inyectar `env[rule.Field] = []valores`
5. Target resolution: match por path exacto, luego fallback a basename (misma logica que graph.go)

**Out**: Topological sort de evaluacion (eso es S002), aggregate changes, UI changes

## Estado inicial esperado

- DeriveAll y DeriveRecord funcionales
- LinkRule.Field parseado en StemFile
- Record.Links populado por MarkdownExtractor

## Criterios de Aceptacion

- DeriveRecord con RecordResolver inyecta valores de campo enlazado en env
- `env["blocked_by"]` contiene slice de valores string del campo referenciado
- Link a target inexistente no produce error (skip silencioso)
- Multiples links del mismo tipo generan slice con todos los valores
- Tests unitarios en `internal/derive/links_test.go` pasan

## Fuente de verdad

- `internal/derive/pipeline.go` (DeriveAll — agregar lookup map)
- `internal/derive/record.go` (DeriveRecord — agregar RecordResolver param)
- `internal/rules/rules.go` (LinkRule struct con Field)
- `internal/graph/graph.go` (resolveTarget — reusar logica de fallback)
