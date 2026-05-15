---
estado: Completed
tipo: task
---
# T003: Add `value_field` to LinkRule

**Outcome**: [O14 Field-agnostic refactor](README.md)
**Contribuye a**: el link enricher no hardcodea "estado" para resolver expresiones sobre links

## Preserva

- INV1: Links sin `value_field` se comportan igual que hoy (no regresión)
  - Verificar: `go test ./internal/derive/... -run TestLink`
- INV2: `all(blocked_by, {# == "Completed"})` sigue evaluando correctamente cuando `value_field: estado` está configurado
  - Verificar: tests de integración existentes

## Contexto

El link enricher en `internal/derive/links.go` inyecta el valor de `"estado"` hardcodeado en el entorno de expresiones cuando resuelve links. Esto impide que rootline sea agnóstico al campo de lifecycle.

El fix es agregar `ValueField string yaml:"value_field" json:"value_field,omitempty"` a `LinkRule` en `internal/rules/rules.go`, y hacer que el enricher inyecte `rule.ValueField` en vez de `"estado"` hardcodeado. Sin `value_field`, no se inyecta ningún campo de valor (opt-in).

## Alcance

**In**:
1. Agregar `ValueField string yaml:"value_field" json:"value_field,omitempty"` a `LinkRule` en `internal/rules/rules.go`
2. Actualizar el link enricher en `internal/derive/links.go` para usar `rule.ValueField` en vez de `"estado"`
3. Sin `value_field`, el enricher no inyecta ningún campo de estado en el entorno de expresiones

**Out**:
- No cambiar la sintaxis de los link references en los archivos .md
- No actualizar el .stem todavía (eso es T002)

## Estado inicial esperado

- `go test ./...` pasa en `/home/shared/rootline`
- `LinkRule` no tiene campo `ValueField`

## Criterios de Aceptación

- `LinkRule` struct tiene `ValueField string yaml:"value_field"`
- Sin `value_field` en el schema, comportamiento idéntico al actual (tests existentes pasan sin cambios)
- Con `value_field: estado`, expresiones como `all(blocked_by, {# == "Completed"})` siguen evaluando correctamente
- `go test ./internal/rules/... ./internal/derive/...` verde

## Fuente de verdad

- `internal/rules/rules.go` — LinkRule struct
- `internal/derive/links.go` — link enricher
