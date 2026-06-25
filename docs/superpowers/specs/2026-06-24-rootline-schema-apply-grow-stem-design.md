# Diseño — `schema apply` hace crecer el `.stem` con campos nuevos

**Fecha:** 2026-06-24
**Estado:** aprobado, pendiente de plan de implementación
**Tipo:** corrección de capacidad (cierra un loop roto)

## Contexto y problema

El flujo de evolución de esquemas en rootline es:

```
analyze --incremental  →  (proposals no cubiertos por el .stem)  →  schema apply
```

`analyze --incremental` y `schema propose --incremental` existen **específicamente** para
reportar inferencias que el `.stem` actual todavía NO cubre — es decir, campos nuevos que
aparecieron en los records después de que se escribió el `.stem`.

Sin embargo, `ApplySchemaInferences` (la función que ejecuta `schema apply`,
`cmd/rootline/schema.go:339`) **solo refina nodos de schema que ya existen**. Cada handler en
`internal/infer/apply.go` resuelve el nodo del campo con `findSchemaFieldNode(doc, field)` y, si el
campo no está en el `.stem`, hace `return false` — descarte silencioso.

**Consecuencia:** el loop `analyze --incremental → schema apply` está roto a la mitad. La primera
parte te muestra el campo nuevo; la segunda no lo puede aplicar. Las dos features se contradicen.

El comportamiento actual está **testeado como intencional** (`TestApplySchemaInferences_EnumNoField`,
`_RequiredFieldNotInSchema`, `_DefaultNotInSchema`, `_FieldTypeNotInSchema` asertan "0 applied").
Por lo tanto, revertirlo es una decisión de diseño consciente, no la corrección de un descuido.

### Decisión

`schema apply` debe **hacer crecer** el `.stem`: cuando una inferencia de schema apunta a un campo
ausente, debe AGREGAR el nodo del campo con la propiedad inferida, en vez de descartarla.
`scaffold` sigue siendo para bootstrap (sin `.stem`); `apply` pasa a cubrir bootstrap + evolución.

### Fuera de scope

- **Governance en islas multi-`.stem`** (los detectores reciben el stem root-merged,
  `cmd/rootline/analyze.go:91-93`): fue una decisión deliberada y documentada, sin driver concreto.
  No se toca.
- Nuevos formatos de extractor (Fase 4): pausado por falta de driver real.

## Arquitectura del cambio

Todo el cambio se concentra en `internal/infer/apply.go`. El `Record` y el pipeline downstream no
se tocan.

### Helper nuevo: `ensureSchemaFieldNode`

Análogo a `findSchemaFieldNode`, pero idempotentemente creador:

```
ensureSchemaFieldNode(doc *yaml.Node, fieldName string) (node *yaml.Node, created bool)
```

- Navega `doc → schema`. Si la key `schema:` no existe (caso de un `.stem` mínimo con solo
  `scope:`), la crea como un `MappingNode` vacío y la agrega a la raíz.
- Busca el campo dentro de `schema:`. Si existe, devuelve `(nodo, false)` (comportamiento actual de
  `findSchemaFieldNode`). Si no existe, agrega `fieldName: {}` (mapping vacío) y devuelve
  `(nodo, true)`.
- Devuelve `(nil, false)` solo si la raíz no es un mapping (`.stem` corrupto), en cuyo caso el
  handler hace `return false` igual que hoy.
- El flag `created` es la fuente única de verdad para distinguir creación de refinamiento (ver
  Observabilidad); los handlers no re-consultan `stem.Schema` para esto.

`findSchemaFieldNode` se conserva sin cambios para los call-sites que no deben crear
(`applySequenceCompleteNode`, que opera sobre un campo de tipo sequence preexistente).

### Cambios por handler

Cada handler pasa de "buscar o rendirse" a "asegurar y poblar". El guard contra `stem.Schema[field]`
(struct parseado al inicio) se mantiene para no re-aplicar sobre estado ya satisfecho, pero deja de
bloquear la creación.

| Inference (`inf.Type`) | Hoy (campo ausente) | Nuevo comportamiento |
|---|---|---|
| `field_type` / `untyped_field` | drop | crea nodo + `type: <inf.Value>` |
| `enum_values` | drop | crea nodo + `type: enum` + `values: [<inferidos>]` |
| `required_field` | drop | crea nodo + `required: true` |
| `constant_field` | drop | crea nodo + `default: <inf.Value>` |
| `sequence_incomplete` | drop | **sin cambio** — sigue requiriendo campo existente |

Detalles:

- **`enum_values` sobre campo nuevo DEBE setear `type: enum`.** Un `values:` sin `type: enum` queda
  huérfano y el validador no lo reconocería como enum. Al crear el nodo, primero `type: enum`, luego
  el `values:` sequence con todos los valores inferidos (para un campo nuevo, `sf.Values` está vacío,
  así que todos los inferidos son nuevos).
- **`required_field` / `constant_field` sobre campo nuevo** setean solo su propiedad. No inventan un
  `type`: si el mismo report trae también un `field_type` para ese campo, el type se setea por esa
  vía. No se asume un type por defecto.

### Ordenamiento dentro de un run

Si un mismo campo nuevo recibe varias inferencias en un solo report (p. ej. `field_type` +
`required_field` + `enum_values`), el `stem` struct parseado en `ApplySchemaInferences` está *stale*:
no refleja nodos agregados durante el loop. Esto es seguro porque, para un campo nuevo,
`stem.Schema[field]` está ausente para TODAS las inferencias del run → cada handler procede y
`ensureSchemaFieldNode` encuentra (o crea, la primera vez) el mismo nodo. Resultado determinista: un
solo nodo de campo con todas las propiedades pobladas.

### Observabilidad

Cuando el campo se **crea** (vs. refinar uno existente), el mensaje en `ApplyResult.Applied` lo
distingue, para que el operador vea que el `.stem` creció:

- Refinar existente: mensajes actuales (`set_type: x=y`, `extend_enum: x`, `add_required: x`, …).
- Crear nuevo: prefijo `add_field` — p. ej. `add_field: x (type=y)`, `add_field: x (enum)`.

La distinción se decide con el flag `created` que devuelve `ensureSchemaFieldNode`.

### Dry-run y round-trip

- `schema apply --dry-run` sigue funcionando: la escritura ya está gateada por `dryRun`; el cambio no
  toca esa ruta.
- El `.stem` resultante debe re-parsear limpio (`rules.ParseStem`) y pasar validación post-apply (el
  comando ya corre validación post-apply).

## Plan de tests (TDD)

### Tests a flipear (codifican el contrato viejo)

Pasan de "0 applied" a "1 applied + campo creado con la propiedad correcta", reescritos RED primero:

- `TestApplySchemaInferences_EnumNoField`
- `TestApplySchemaInferences_RequiredFieldNotInSchema`
- `TestApplySchemaInferences_DefaultNotInSchema`
- `TestApplySchemaInferences_FieldTypeNotInSchema`

### Tests nuevos

- `field_type` sobre campo ausente → crea `schema.<field>.type`.
- `enum_values` sobre campo ausente → crea `schema.<field>` con `type: enum` + `values`.
- `.stem` sin key `schema:` → la crea y agrega el campo.
- Múltiples inferencias para el mismo campo nuevo en un run → un solo nodo, todas las propiedades.
- Round-trip: el `.stem` escrito re-parsea con `rules.ParseStem` sin error y es bien-formado.
- Mensajes de `ApplyResult.Applied` distinguen `add_field` (creado) de los refinamientos existentes.

## Verificación (Definition of Done)

- `just check` (gofmt + golangci-lint + build) en verde.
- `just test` (con `-race`) en verde, 0 failures.
- Cobertura `internal/infer/` ≥ 85% (`.coverage-floors.toml`).
- Documentar la capacidad nueva en `docs/extensibility.md`/README solo si corresponde (el contrato de
  `schema apply` cambia: ahora puede crear campos).
- Commit convencional + push; CLI instalado y `rootline --version` verificado.

## Commit

`feat(schema): grow .stem with newly-observed fields on apply`

Nueva capacidad: cierra el loop `analyze --incremental → schema apply`. Pre-1.0 → bump patch.
