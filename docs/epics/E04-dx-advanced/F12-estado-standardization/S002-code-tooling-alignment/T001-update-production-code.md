---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Update infer.go and links.go production code

**Story**: [S002 Code & Tooling Alignment](README.md)

## Contexto

Dos archivos de produccion en Go tienen valores de estado hardcodeados en espanol y un bug en la lectura de estado de records enlazados.

`internal/proposal/infer.go` tiene `valueMapping` que mapea "completada" → "Completado" y una funcion `InferEstado` que retorna "Completado". Ambos deben usar los nuevos valores en ingles.

`internal/derive/links.go` linea 129 usa `target.Frontmatter["estado"]` directamente, ignorando valores derivados. Debe usar `target.EffectiveField("estado")` que prioriza Derived sobre Frontmatter. Los doc comments tambien referencian valores en espanol.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/proposal, internal/derive
interfaces:
  - nombre: valueMapping (var)
    metodos:
      - nombre: mapValue
        input: "val string"
        output: "string"
  - nombre: InferEstado
    metodos:
      - nombre: InferEstado
        input: "childEstados []string"
        output: "string"
  - nombre: InjectLinkedFields
    metodos:
      - nombre: InjectLinkedFields
        input: "env map[string]any, record *extract.Record, stem *rules.StemFile, resolver RecordResolver"
        output: ""
dependencias_externas: []
tests:
  - mapValue("completada") retorna "Completed"
  - InferEstado(["Completed", "Completed"]) retorna "Completed"
  - InjectLinkedFields usa EffectiveField en vez de Frontmatter directo
```

## Alcance

**In**:
1. `internal/proposal/infer.go`: Cambiar `"Completado"` → `"Completed"` en valueMapping y InferEstado
2. `internal/derive/links.go` linea 129: Cambiar `target.Frontmatter["estado"]` → usar `target.EffectiveField("estado")`
3. `internal/derive/links.go` doc comments: Actualizar referencias a "Completado", "Bloqueada", "Pending" → "Completed", "Blocked", "Pending"

**Out**: Test files (T002), skill files (T003), .stem changes (S001)

## Estado inicial esperado

- S001 completado (schema migrado, frontmatter migrado)
- `infer.go` tiene `"completada": "Completado"` en valueMapping
- `links.go` tiene `target.Frontmatter["estado"]` en linea 129

## Criterios de Aceptacion

- `grep "Completado" internal/proposal/infer.go` retorna 0 lineas
- `grep 'Frontmatter\["estado"\]' internal/derive/links.go` retorna 0 lineas
- `grep "Completado" internal/derive/links.go` retorna 0 lineas
- `go build ./...` compila sin errores
- `go test ./internal/proposal/ -run TestInfer -v` pasa
- `go test ./internal/derive/ -run TestInject -v` pasa

## Fuente de verdad

- `internal/proposal/infer.go`
- `internal/derive/links.go`
- `internal/extract/extract.go` (EffectiveField method)
