---
estado: Specified
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T005: Implement validate conditional with match pattern

## Descripción

El research doc (Part 3, tabla línea 157) propone que per-level `validate:` rules se reemplacen con condicionales basadas en directory pattern:

```yaml
validate:
  - rule: requires
    if: { match: "T*", tipo: "software-module" }
    then: { fields: [ejecutable_en] }
```

Actualmente `conditionMatchesRecord()` solo compara campos del record contra valores. Debe reconocer `match` como clave especial que compara contra el directory name del record.

## Antes / Después

**Antes**: `validate: if: {tipo: "software-module"}` aplica en todos los niveles. No hay forma de limitar una regla de validación a un patrón de directorio.

**Después**: `validate: if: {match: "T*", tipo: "software-module"}` aplica solo en directorios que matchean `T*`. La validación no se dispara para records en `F*` dirs aunque tengan `tipo: software-module`.

## Criterios de Aceptación

- [ ] `conditionMatchesRecord()` reconoce `match` como clave especial
- [ ] Si `match` está en `if:`, compara `filepath.Base(filepath.Dir(record.Path))` contra el pattern
- [ ] Regla con `if: {match: "T*", tipo: X}` → solo aplica en T* dirs con tipo == X
- [ ] Regla con `if: {match: "T*"}` sin otros campos → aplica a todos los T* dirs
- [ ] `ValidationRule` struct no cambia — `If map[string]any` ya acepta `match`
- [ ] Migrador inyecta `match: "PATTERN"` en `if:` clause de per-level validate rules

## Preserva

- INV1: Reglas existentes sin `match` en `if:` siguen funcionando exactamente igual
- INV2: Tests de validate existentes no se rompen

## Dependencias

- [[blocks:T002-implement-match-aware-field-resolution]] (FilterSchemaByMatch debe existir para contexto)
