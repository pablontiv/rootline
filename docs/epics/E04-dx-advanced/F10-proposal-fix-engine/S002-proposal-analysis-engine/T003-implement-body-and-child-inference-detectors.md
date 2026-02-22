---
estado: Pending
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Implement Body Extraction and Child Inference Detectors

**Story**: [S002 Proposal Analysis Engine](README.md)

[[blocks:T001-create-proposal-types-and-basic-detectors]]
[[blocks:T001-add-scan-body-fields]]

## Contexto

Dos tipos de propuesta requieren analisis mas alla de los errores de validacion:

1. **extract_body**: Cuando un archivo no tiene frontmatter pero su body contiene `**Estado**: Completada`, rootline deberia proponer extraer ese valor a YAML frontmatter. Usa `extract.ScanBodyFields()` de S001/T001. Necesita value mapping porque los valores en body pueden diferir del enum (ej: "Completada" → "Completado", "Activa" → "In Progress").

2. **infer_from_children**: Cuando un README de directorio no tiene estado en frontmatter ni en body, rootline deberia inferirlo de los archivos hijos. Si todos los hijos son Completado → el padre es Completado. Si hay mix → In Progress. Si todos Pending → Pending.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/proposal
interfaces:
  - nombre: detectExtractBody
    metodos:
      - nombre: detectExtractBody
        input: "requiredErrors []requiredError, records []*extract.Record, schema *rules.StemFile"
        output: "[]Proposal"
  - nombre: detectInferFromChildren
    metodos:
      - nombre: detectInferFromChildren
        input: "requiredErrors []requiredError, records []*extract.Record, schema *rules.StemFile"
        output: "[]Proposal"
  - nombre: InferEstado
    metodos:
      - nombre: InferEstado
        input: "childEstados []string"
        output: "string"
dependencias_externas: []
tests:
  - Record con body "**Estado**: Completada" → extract_body proposal, mapped "Completado"
  - Record con body "**Estado**: Activa" → extract_body proposal, mapped "In Progress"
  - README.md con hijos todos Completado → infer_from_children "Completado"
  - README.md con hijos mixed → infer_from_children "In Progress"
  - README.md con hijos todos Pending → infer_from_children "Pending"
  - InferEstado([]) → "Pending" (no children, default)
```

## Alcance

**In**:
1. Crear `internal/proposal/infer.go` con `InferEstado()` y value mapping table
2. Value mapping: `{"completada": "Completado", "activa": "In Progress", "activo": "In Progress", "pendiente": "Pending"}` — case-insensitive
3. Agregar `detectExtractBody()` en `detect.go` — usa `extract.ScanBodyFields()` en records con required errors
4. Agregar `detectInferFromChildren()` — para READMEs sin body hints, busca records hijos por path prefix
5. Integrar en `Analyze()` — extract_body prioridad 3, infer_from_children prioridad 4
6. Tests

**Out**: No eliminar el patron `**Key**: Value` del body al aplicar — eso es decision de S003/T002. Solo detectar y proponer.

## Estado inicial esperado

- `internal/proposal/` existe con tipos y detectores basicos (de T001)
- `internal/extract/extract.go` tiene `ScanBodyFields()` (de S001/T001)
- `Analyze()` funciona para extend_enum, migrate_value, correct_value, add_field

## Criterios de Aceptacion

- `go test ./internal/proposal/ -run TestDetectExtractBody -v` pasa: body con `**Estado**: Completada` → proposal con mapped value "Completado"
- `go test ./internal/proposal/ -run TestInferEstado -v` pasa: [Completado, Completado] → "Completado"; [Pending, Completado] → "In Progress"
- `go test ./internal/proposal/ -run TestDetectInferFromChildren -v` pasa: README con hijos mixtos → infer proposal
- `go test ./internal/proposal/ -run TestAnalyze -v` pasa con todos los proposal types integrados
- `go vet ./internal/proposal/` sin errores

## Fuente de verdad

- `internal/proposal/infer.go` — archivo nuevo
- `internal/proposal/detect.go` — extender con 2 detectores
- `internal/extract/extract.go` — ScanBodyFields()
- `/opt/homeserver/automation/docs/epics/E01-infrastructure-foundation/README.md` — ejemplo real de body con `**Estado**: Completada`
