---
estado: Completed
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T002: Integrar PR annotations via workflow commands

**Story**: [S001 Validation Action](README.md)

[[blocks:T001-composite-action]]

## Contexto

La action de T001 ejecuta rootline validate y obtiene JSON output. Esta task parsea ese JSON y emite GitHub workflow commands (`::error file=<path>::<message>`) para que los errores aparezcan como annotations directamente en el diff del PR.

## Especificacion Tecnica

```yaml
tipo: ci-cd
plataforma: github-actions
triggers:
  - pr
jobs:
  - nombre: annotate
    pasos:
      - Ejecutar rootline validate --all --output json
      - Parsear BatchValidationResult JSON
      - Para cada error, emitir ::error file=<path>::<message>
artefactos:
  - Script de parseo integrado en action.yml
```

## Dependencias

- T001 completado (action definition con validate funcional)

## Alcance

**In**:
1. Parsear JSON output de `rootline validate --all --output json`
2. Emitir `::error file=<path>::<message>` por cada error de validacion
3. Emitir `::warning file=<path>::<message>` por cada warning (severity=warn)
4. Summary output con conteo de errores/warnings

**Out**: Review comments via Checks API, reviewdog integration, rich markdown summaries

## Estado inicial esperado

- Action composite funcional (T001)
- rootline validate --output json produce BatchValidationResult

## Criterios de Aceptacion

- PR con documento invalido muestra annotation de error en la linea del archivo
- PR con warning (severity=warn) muestra annotation de warning
- Documento valido no produce annotations
- Step summary muestra conteo total de errores y warnings

## Fuente de verdad

- `internal/rules/result.go` (BatchValidationResult struct — campos path, errors, message)
- GitHub workflow commands docs (::error, ::warning syntax)
