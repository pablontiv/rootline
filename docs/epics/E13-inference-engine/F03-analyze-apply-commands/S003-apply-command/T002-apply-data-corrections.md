---
estado: Specified
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar aplicacion de correcciones de datos

**Story**: [S003 Apply Command](README.md)
**Contribuye a**: `rootline apply` corrige datos en documentos basado en inferencias

## Preserva

- INV1: `go test ./... -race` pasa verde
  - Verificar: `go test ./... -race`
- INV3: Coverage ≥85%
  - Verificar: `go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`

## Contexto

Ademas de modificar .stem (T001), apply puede corregir datos en documentos: migrate_value (renombrar valor de campo), correct_value (corregir typo detectado), add_field (añadir campo missing con default). internal/fix/ ya implementa estas operaciones para propuestas de validacion.

## Alcance

**In**:
1. Extender apply.go para manejar inferencias de tipo data-correction:
   - `migrate_value` → cambiar valor de campo en frontmatter de documentos afectados
   - `correct_value` → corregir valor (typo, casing)
   - `add_field` → añadir campo con default value a documentos que no lo tienen
2. Reusar logica de internal/fix/ para rewrite de frontmatter
3. Dry-run mode: `--dry-run` muestra cambios sin aplicar

**Out**: Correcciones de body content (no se modifica body). Correcciones que necesitan agent.

## Estado inicial esperado

- T001 completado (apply command existe con schema mods)
- internal/fix/fix.go tiene ApplyProposals

## Criterios de Aceptacion

- apply con inferencia `migrate_value{field: "estado", from: "todo", to: "Pending"}` cambia valor en documentos
- `--dry-run` muestra cambios sin modificar archivos
- Documentos modificados pasan `rootline validate`
- `go test ./... -race` pasa verde

## Fuente de verdad

- `internal/fix/fix.go` — ApplyProposals, rewrite logica
- `internal/proposal/proposal.go` — Proposal types
