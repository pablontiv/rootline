---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Implementar aplicacion de modificaciones de schema

**Story**: [S003 Apply Command](README.md)
**Contribuye a**: `rootline apply` modifica .stem basado en inferencias

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV2: .stem modificados siguen siendo v2 validos
  - Verificar: `rootline validate <modified-stem-dir>`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

El analyze report contiene inferencias que sugieren cambios al .stem: extend_enum, add_required_field, constant_field → default value. internal/fix/ ya tiene logica para rewrite de frontmatter en documentos. Apply necesita logica similar para .stem files.

## Alcance

**In**:
1. Crear `cmd/rootline/apply.go` con cobra command
2. Leer report JSON de stdin o archivo
3. Para cada inferencia de tipo schema-modification:
   - `extend_enum` → añadir valor al campo enum en .stem
   - `add_required_field` → añadir campo a lista required en .stem
   - `add_default` → añadir default value para campo constante
4. Escribir .stem modificado con formato YAML preservado
5. Saltar inferencias con `requires_agent: true` con warning

**Out**: Correcciones de datos en documentos (T002). UI interactiva (futuro).

## Estado inicial esperado

- S001 completado (analyze produce report JSON)
- internal/fix/ tiene logica de rewrite de YAML

## Criterios de Aceptacion

- `rootline apply report.json` modifica .stem file
- extend_enum: stem antes tiene `enum: [a, b]` → despues `enum: [a, b, c]`
- requires_agent inferencias producen warning pero no aplican cambios
- .stem resultante pasa `rootline validate`
- `go test ./... -race` pasa verde

## Fuente de verdad

- `internal/fix/fix.go` — logica de rewrite de YAML
- `internal/rules/rules.go` — StemSchema para leer/escribir .stem
