# Diseño — `validate` reporta errores de YAML malformado en vez de tragarlos

**Fecha:** 2026-06-25
**Estado:** aprobado, pendiente de plan de implementación
**Tipo:** corrección de capacidad (cierra un gap de validación)

## Contexto y problema

Cuando el frontmatter YAML de un record no parsea, rootline lo detecta pero **no lo
reporta como falla de `validate`**. La cadena (`internal/extract/extract.go:116-125`):

```go
if err := yaml.Unmarshal([]byte(fmContent), &record.Frontmatter); err != nil {
    record.Errors = append(record.Errors, ExtractionError{...})  // no-fatal
    record.Frontmatter = fallbackParseFrontmatter(fmContent)      // fallback permisivo
}
```

Al fallar el parser estricto (`gopkg.in/yaml.v3`), rootline (1) guarda el error como un
`ExtractionError` **no-fatal** en `record.Errors`, y (2) cae a un parser línea-por-línea
permisivo (`fallbackParseFrontmatter`) que rescata el frontmatter igual.

Después, `cmd/rootline/validate.go` recibe `(record, nil)` —el parse error vive en
`record.Errors`, no en el `err` retornado— y **nunca lee `record.Errors`**:

- `runValidateFiles` (validate.go:87-107): corre `rules.Validate` + `ValidateStructure`; nunca
  toca `record.Errors`.
- `runValidateAll` (validate.go:171-186): corre `rules.Validate` sobre los records del index
  scanner; nunca toca `rec.Errors`. (El index sí preserva `rec.Errors`: index.go:165-170
  guarda el `rec` completo y `extractErr` es nil para YAML malformado.)

**Consecuencia:** un frontmatter YAML malformado **pasa `validate`** pero rompe cualquier
parser estricto aguas abajo (js-yaml/Quartz). Esto causó un build de Quartz caído en el
proyecto wiki por `title:` con un `:` interno sin comillar; `rootline validate --all` no lo
cacheó.

**Aclaración importante:** el parser YAML de rootline NO es permisivo — `yaml.v3` rechaza el
mismo input que js-yaml (verificado empíricamente: `title: A: B` → error
`mapping values are not allowed in this context`). La permisividad está en el **fallback** y en
que el error **se descarta** en la capa de validate, no en el parseo.

### Decisión

`validate` debe **reportar** los `record.Errors` (parse errors de YAML) como errores bloqueantes,
en ambas rutas (single-file y `--all`). Comportamiento elegido: **falla por defecto + reporta
todo** — emite el error `malformed_yaml` Y además sigue corriendo el schema validation sobre el
frontmatter rescatado por el fallback (panorama completo, aunque más ruidoso).

### Fuera de scope

- **Remover el fallback parser:** se mantiene; los read-paths (`query`, `tree`, `graph`) deben
  seguir resilientes ante YAML roto, no romperse.
- **Flag `--strict`:** no se agrega; el error es bloqueante por defecto, consistente con
  `multiple_yaml_documents`.
- **`query`/`tree`/`fix`/`set`/`explain`:** no cambian; el único comando que debe FALLAR ante
  YAML malformado es `validate`.
- **Hallazgo adyacente (anotado, NO se construye):** `runValidateAll` no corre `ValidateStructure`
  per-file, así que `multiple_yaml_documents` tampoco se cachea en `--all`. Gap distinto, sin
  driver hoy.

## Arquitectura del cambio

Precedente exacto: `rules.ValidateStructure` (validate.go:413) ya inyecta un error estructural
(`multiple_yaml_documents`, severity `error`) desde contenido crudo, separado del schema, y ya
está wireado en `runValidateFiles`. El error de YAML malformado encaja en el mismo molde.

### Helper nuevo: `rules.ExtractionErrors`

```go
// ExtractionErrors converts a record's non-fatal extraction errors (e.g. malformed
// YAML frontmatter) into blocking ValidationErrors so validate surfaces them.
func ExtractionErrors(rec *extract.Record) []ValidationError
```

- Itera `rec.Errors` (`[]extract.ExtractionError`, campos `Line`, `Message`).
- Por cada uno emite:
  ```go
  ValidationError{
      Rule:       "malformed_yaml",
      Field:      "_frontmatter",
      Message:    ee.Message,        // p.ej. "malformed YAML frontmatter: yaml: mapping values are not allowed in this context"
      Source:     rec.Path,
      Severity:   "error",
      Suggestion: "quote values containing ':' or other YAML-special characters",
  }
  ```
- `rec.Errors` vacío → devuelve `nil`.
- `rules` ya importa `extract.Record` (lo recibe `rules.Validate`), así que no hay ciclo de
  imports.

### Wire en las dos rutas de `validate`

- **`runValidateFiles`** (validate.go:~104, junto a `structErrs`):
  ```go
  errs = append(errs, rules.ExtractionErrors(record)...)
  ```
- **`runValidateAll`** (validate.go:~178, junto a `rules.Validate`):
  ```go
  errs = append(errs, rules.ExtractionErrors(rec)...)
  ```

### Sin cambios

- `internal/extract/extract.go` y `fallbackParseFrontmatter`: intactos.
- El schema validation sigue corriendo sobre el frontmatter rescatado (decisión "reportar todo"):
  un archivo malformado puede producir 1 `malformed_yaml` + N errores de schema.

### Contrato / observabilidad

- El error sale como cualquier `ValidationError` (Rule `malformed_yaml`) en la salida JSON/table de
  `validate`; el JSON mantiene `version: 1` (cambio aditivo: un nuevo valor de `Rule`, sin cambio
  de esquema).
- `validateHasFailure` ya trata cualquier `ValidationError` con severity `error` como falla → exit
  code 1 automático.

## Plan de tests

### Unit (`internal/rules`)

- `ExtractionErrors` sobre un record con un `ExtractionError` → un `ValidationError` con Rule
  `malformed_yaml`, Severity `error`, Source = rec.Path.
- `ExtractionErrors` sobre un record sin errores → `nil`.

### Integración (`cmd/rootline` / e2e)

- Fixture con `title:` con `:` interno sin comillar → `validate <file>` reporta `malformed_yaml` y
  exit code 1.
- Mismo fixture vía `validate --all` → reporta `malformed_yaml` (cubre la ruta del index scanner,
  que es la que usa el wiki).
- "Reportar todo": fixture malformado cuyo campo rescatado por el fallback viola el schema →
  aparecen AMBOS (`malformed_yaml` + el error de schema).
- Regresión: fixture válido → sin `malformed_yaml`.

## Verificación (Definition of Done)

- `just check` (gofmt + golangci-lint + build) en verde.
- `just test` (con `-race`) en verde, 0 failures.
- Cobertura `internal/rules` y `cmd/rootline` ≥ 85% (`.coverage-floors.toml`).
- `rootline validate --all docs/epics` (lo que corre el CI propio del repo) sigue pasando — los
  docs del repo no deben regresionar por el nuevo error.
- Commit convencional + push; CLI instalado y `rootline --version` verificado.

## Commit

`feat(validate): surface malformed YAML frontmatter as a blocking error`

Cierra el gap donde YAML malformado pasaba `validate` (parse error tragado). El bump de versión lo
determina el CI por tipo de commit. Es stricter-by-default: archivos con YAML roto que hoy pasan,
ahora fallan validate — el equipo decide si el commit lleva `!` (breaking) según su política; este
spec no lo prejuzga.
