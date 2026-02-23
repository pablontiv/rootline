---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T004: extract_body/infer_from_children tienen prioridad sobre add_field

**Story**: [S001 Fix Priority Conflicts](README.md)

## Contexto

`detectAddField` genera proposals con valor default ("Pending") para READMEs que faltan el campo `estado`. Pero `detectExtractBody` e `detectInferFromChildren` ya generan proposals con valores correctos (extraidos del body o derivados de hijos) para esos mismos archivos.

Al aplicar, el orden de ejecucion determina quien gana. Si `add_field` se aplica ultimo, sobreescribe el valor correcto con "Pending".

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
archivo: internal/proposal/proposal.go
funcion: Analyze
cambio: |
  1. Detectar extract_body e infer_from_children primero
  2. Construir set de path+field cubiertos
  3. Filtrar add_field: skip si path+field ya cubierto
tests:
  - 0 add_field proposals para READMEs con extract_body
  - 0 add_field proposals para READMEs con infer_from_children
  - add_field SI se genera para archivos sin body hint ni hijos
```

## Alcance

**In**: Filtrar add_field en Analyze por cobertura de otros detectores
**Out**: Cambios en detectAddField, detectExtractBody o detectInferFromChildren internamente

## Estado inicial esperado

- READMEs sin frontmatter existen con estados en body (bold-colon pattern)
- READMEs sin frontmatter existen con hijos que tienen estados

## Criterios de Aceptacion

- `rootline fix --all --dry-run -o json | jq '.proposals[] | select(.type=="add_field")' | wc -l` retorna solo archivos sin cobertura de otros detectores
- READMEs con body hints reciben `extract_body`, no `add_field`
- READMEs con hijos reciben `infer_from_children`, no `add_field`
- `go test ./internal/proposal/...` pasa

## Fuente de verdad

- `internal/proposal/proposal.go` funcion Analyze
