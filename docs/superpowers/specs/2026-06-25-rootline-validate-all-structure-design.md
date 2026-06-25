# Diseño — `validate --all` corre el chequeo estructural por archivo

**Fecha:** 2026-06-25
**Estado:** aprobado, pendiente de plan de implementación
**Tipo:** corrección de paridad (cierra un gap entre validate single-file y `--all`)

## Contexto y problema

`rules.ValidateStructure(content, path)` detecta archivos con múltiples documentos YAML
(`multiple_yaml_documents`, severity `error`) contando los separadores `---` en el contenido crudo.

`runValidateFiles` (single-file) ya lo corre (`cmd/rootline/validate.go:100,105`). Pero
`runValidateAll` (`--all`) **no**: su loop per-record (validate.go:171-186) trabaja con los
`*extract.Record` del index scanner, que NO retienen el contenido crudo, así que el chequeo
estructural nunca corre en `--all`. Resultado: un archivo con múltiples docs YAML pasa
`validate --all` pero falla `validate <file>`. Paridad rota.

Es el gap hermano del que cerramos en
`2026-06-25-rootline-validate-surface-yaml-errors-design.md` (que ya agregó `ExtractionErrors` a
ambas rutas); este completa la paridad para el chequeo estructural.

### Decisión

`runValidateAll` debe correr `ValidateStructure` por record. Como el `Record` no retiene el
contenido crudo, se **re-lee el archivo** en el loop (`os.ReadFile(absPath)`), espejando el
single-file. Mínimo blast radius (solo `validate.go`); el costo de una lectura extra por archivo es
aceptable (validate no es perf-crítico). Descartado: agregar `RawContent` al `Record` (toca el
extractor y todos los consumidores; optimización sin driver de perf).

### Fuera de scope

- Cambiar el `extract.Record` o el extractor.
- Las rutas de read (`query`/`tree`/`graph`), que no deben fallar.

## Arquitectura del cambio

Todo en `cmd/rootline/validate.go`, dentro del loop per-record de `runValidateAll` (~línea 178,
junto a los appends de `rules.Validate` y `rules.ExtractionErrors`):

```go
errs := rules.Validate(ctx, rec, effective)
errs = append(errs, rules.ExtractionErrors(rec)...)
if content, readErr := os.ReadFile(absPath); readErr == nil {
    errs = append(errs, rules.ValidateStructure(content, rec.Path)...)
}
```

- `absPath` ya está computado en el loop. `os` ya está importado (lo usa `runValidateFiles`).
- Si `readErr != nil` (archivo borrado entre el scan y ahora): se omite el chequeo estructural; el
  record igual valida por las otras vías. Degradación silenciosa aceptable.

## Plan de tests (integración, `cmd/rootline`)

- Fixture con múltiples docs YAML (`---\nfoo: 1\n---\nbody\n---\nbar: 2\n---\n`) en un dir temporal
  → `validate --all <dir>` reporta `multiple_yaml_documents` y exit code 1.
- Regresión: fixture de un solo doc → `validate --all` no reporta `multiple_yaml_documents`.

## Verificación (Definition of Done)

- `just check` (gofmt + golangci-lint + build) en verde.
- `just test` (`-race`) en verde, 0 failures.
- Cobertura `cmd/rootline` ≥ 85% (`.coverage-floors.toml`).
- Commit convencional + push.

## Commit

`fix(validate): run structural check per-file in --all mode`

Cierra la paridad: `multiple_yaml_documents` ahora se caza en `--all` igual que en single-file.
