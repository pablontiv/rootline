---
estado: Completed
tipo: modulo-sistema
ejecutable_en: 1 sesion
---
# T004: Implement required scopeable by match

## Descripción

El research doc propone que `required` no sea un booleano global, sino scopeable por directory pattern:

```yaml
tipo:
  type: enum
  match: ["F*", "T*"]
  required:
    match: ["T*"]    # required solo a nivel task
```

Esto permite que un campo sea obligatorio solo en ciertos niveles de la jerarquía sin duplicar la definición del campo.

## Antes / Después

**Antes**: `required: true` aplica universalmente. Para tener `tipo` required solo en tasks, hay que duplicar el campo en `levels:` con diferente `required` por nivel (v1 approach).

**Después**: `required: {match: ["T*"]}` hace que `tipo` sea required solo en directorios que matchean `T*`. `FilterSchemaByMatch` resuelve `RequiredMatch` al filtrar: si el dirName matchea, `Required = true`.

## Criterios de Aceptación

- [ ] `SchemaField.RequiredMatch` parseable desde YAML (`yaml:"required_match"` o custom UnmarshalYAML)
- [ ] `required:` acepta tanto `true`/`false` como `{match: ["T*"]}`
- [ ] `FilterSchemaByMatch` resuelve `RequiredMatch` → `Required = true` cuando dirName matchea
- [ ] Validate reporta error para campo faltante en T* dir, no error en F* dir
- [ ] `MarshalStemV2` serializa `required: {match: [...]}` correctamente
- [ ] Migrador (`ConvertLevelsToMatch`) genera `required: {match: [...]}` cuando required varía por nivel

## Preserva

- INV1: Backward compat — `required: true` sigue funcionando como bool
- INV2: Tests existentes no se rompen

## Dependencias

- [[blocks:T002-implement-match-aware-field-resolution]] (FilterSchemaByMatch debe existir primero)
