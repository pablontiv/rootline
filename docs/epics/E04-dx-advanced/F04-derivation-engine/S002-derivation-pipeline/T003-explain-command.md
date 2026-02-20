---
estado: Diferida
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Implementar comando explain con tracing completo

**Story**: [S002 Derivation Pipeline](README.md)

## Contexto

El stub `cmd/rootline/explain.go` existe y solo imprime "not implemented yet". Con el pipeline de derivacion funcional, explain puede ahora mostrar para cada campo de un documento: su valor actual, su origen (.stem source), si es de schema/derive/validate, y la expresion evaluada si es derivado. Es el comando de observabilidad del sistema.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: explainCmd
    metodos:
      - nombre: RunE
        input: "cmd *cobra.Command, args []string"
        output: "error"
dependencias_externas: []
tests:
  - explain muestra campos de frontmatter con source .stem
  - explain muestra campos derivados con expresion
  - explain muestra errores de validacion con regla source
  - explain -o json produce ExplainResult versionado
```

## Dependencias

- T001, T002 (derivation pipeline funcional)
- internal/rules (Validate, Describe)
- internal/derive (DeriveRecord)

## Alcance

**In**:
1. Reemplazar stub con implementacion completa
2. Argumento requerido: `rootline explain <file>`
3. Para cada campo del documento:
   - Valor actual
   - Origen: "schema" (de .stem), "frontmatter" (del archivo), "derived" (computado)
   - Source .stem file path
   - Expresion (si derivado)
4. Para errores de validacion: regla, campo, mensaje, source
5. Output JSON con `version: 1, kind: "rootline/explain"`
6. Output table con formato legible

**Out**: Explain de cambios entre versiones, explain interactivo, explain de multiples archivos

## Estado inicial esperado

- cmd/rootline/explain.go con stub existente
- Derivation pipeline funcional (T001, T002)
- Validate y Describe funcionales

## Criterios de Aceptacion

- `rootline explain docs/epics/E03-rootline/F01-project-foundation/S001-repository-scaffold/T001-init-go-module.md` muestra campos con origen
- `rootline explain -o json` produce JSON con version:1 y kind:"rootline/explain"
- Campos derivados muestran expresion que los genero
- Errores de validacion muestran regla y source .stem
- `go build ./cmd/rootline/` compila sin errores

## Fuente de verdad

- `cmd/rootline/explain.go` — stub existente (reemplazar)
- `internal/rules/describe.go` — DescribeResult, NewDescribeResult
- `internal/rules/validate.go` — Validate
- `internal/derive/` — DeriveRecord
