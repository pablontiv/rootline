---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: migrate_value tiene prioridad sobre correct_value

**Story**: [S001 Fix Priority Conflicts](README.md)

## Contexto

`detectMigrateValue` y `detectCorrectValue` procesan los mismos errores de enum independientemente. Para un archivo con `estado: Pending (blocked by T001)`, ambos generan un proposal:
- `migrate_value`: "Pending (blocked by T001)" -> "Bloqueada" + wiki-links `[[blocks:T001]]`
- `correct_value`: "Pending (blocked by T001)" -> "Pending" (closest match, sin wiki-links)

Si `correct_value` se aplica despues, sobreescribe el valor correcto y pierde los wiki-links.

Adicionalmente, `extend_enum` propone agregar valores que son candidatos a migracion (como "Pending (blocked by T001)") al enum del .stem, lo cual es incorrecto — estos valores deben migrarse, no extenderse.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
archivo: internal/proposal/proposal.go
funcion: Analyze
cambio: |
  1. Detectar migrate_value ANTES de extend_enum y correct_value
  2. Construir set de valores que son candidatos a migracion (p.From)
  3. Filtrar extend_enum: skip si p.Value esta en migrateValues
  4. Construir set de path+field cubiertos por migrate_value
  5. Filtrar correct_value: skip si path+field ya cubierto por migrate_value
tests:
  - 0 correct_value proposals para archivos con migrate_value
  - 0 extend_enum proposals para valores con parentesis (candidatos a migracion)
  - migrate_value proposals preservan wiki-links
```

## Alcance

**In**: Reordenar y filtrar detectores en Analyze
**Out**: Cambios en detectMigrateValue o detectCorrectValue internamente

## Estado inicial esperado

- Archivos con `estado: Pending (blocked by X)` existen en docs/epics/
- Ambos detectores generan proposals para esos archivos

## Criterios de Aceptacion

- `rootline fix --all --dry-run -o json` no muestra `correct_value` para paths que tienen `migrate_value`
- `rootline fix --all --dry-run -o json` no muestra `extend_enum` para valores con parentesis
- `migrate_value` proposals tienen campo `wiki_links` no vacio
- `go test ./internal/proposal/...` pasa

## Fuente de verdad

- `internal/proposal/proposal.go` funcion Analyze
